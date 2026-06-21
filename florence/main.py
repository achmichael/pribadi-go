"""
Florence-2-large-ft – FastAPI inference server
------------------------------------------------
Endpoints:
  POST /analyze          – full image analysis (auto-detect best task)
  POST /caption          – short caption
  POST /detailed-caption – paragraph caption
  POST /ocr              – extract text from image
  POST /object-detection – detect objects, return labels + bounding boxes
  GET  /health           – liveness probe
"""

from __future__ import annotations

import io
import logging
from contextlib import asynccontextmanager
from typing import Any

# pyrefly: ignore [missing-import]
import torch
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.responses import JSONResponse
from PIL import Image
from transformers import AutoModelForCausalLM, AutoProcessor

# ── Logging ───────────────────────────────────────────────────────────────────
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger(__name__)

# ── Model config ──────────────────────────────────────────────────────────────
MODEL_ID = "microsoft/Florence-2-large-ft"
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"
DTYPE = torch.float16 if DEVICE == "cuda" else torch.float32

# ── Global model state (loaded once at startup) ───────────────────────────────
_model: AutoModelForCausalLM | None = None
_processor: AutoProcessor | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load model on startup, release on shutdown."""
    global _model, _processor

    log.info("Loading Florence-2-large-ft on device=%s dtype=%s", DEVICE, DTYPE)
    
    # --- Monkey Patch transformers dynamic imports ---
    # Workaround for the ImportError caused by 'flash_attn' in the downloaded modeling file
    import transformers.dynamic_module_utils
    import os
    
    _original_get_imports = transformers.dynamic_module_utils.get_imports
    
    def _patched_get_imports(filename: str | os.PathLike) -> list[str]:
        imports = _original_get_imports(filename)
        if "flash_attn" in imports:
            imports.remove("flash_attn")
        return imports
        
    transformers.dynamic_module_utils.get_imports = _patched_get_imports
    # -------------------------------------------------

    _processor = AutoProcessor.from_pretrained(MODEL_ID, trust_remote_code=True)
    _model = AutoModelForCausalLM.from_pretrained(
        MODEL_ID,
        torch_dtype=DTYPE,
        trust_remote_code=True,
        attn_implementation="eager"
    ).to(DEVICE)
    _model.eval()
    log.info("Florence-2 ready.")

    yield  # server is live

    log.info("Shutting down – releasing model.")
    del _model, _processor
    if DEVICE == "cuda":
        torch.cuda.empty_cache()


app = FastAPI(
    title="Florence-2 Inference API",
    version="1.0.0",
    lifespan=lifespan,
)


# ── Helper ────────────────────────────────────────────────────────────────────
def _infer(image: Image.Image, task_prompt: str, text_input: str | None = None) -> Any:
    """Run Florence-2 inference for a given task prompt."""
    prompt = task_prompt if text_input is None else f"{task_prompt} {text_input}"

    inputs = _processor(text=prompt, images=image, return_tensors="pt").to(DEVICE, DTYPE)

    with torch.inference_mode():
        generated_ids = _model.generate(
            input_ids=inputs["input_ids"],
            pixel_values=inputs["pixel_values"],
            max_new_tokens=1024,
            num_beams=3,
            do_sample=False,
        )

    generated_text = _processor.batch_decode(generated_ids, skip_special_tokens=False)[0]
    parsed = _processor.post_process_generation(
        generated_text,
        task=task_prompt,
        image_size=(image.width, image.height),
    )
    return parsed


def _load_image(file_bytes: bytes) -> Image.Image:
    try:
        return Image.open(io.BytesIO(file_bytes)).convert("RGB")
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"Invalid image: {exc}") from exc


def _check_ready():
    if _model is None or _processor is None:
        raise HTTPException(status_code=503, detail="Model not loaded yet")


# ── Routes ────────────────────────────────────────────────────────────────────

@app.get("/health")
async def health():
    return {
        "status": "ok",
        "device": DEVICE,
        "model": MODEL_ID,
        "ready": _model is not None,
    }


@app.post("/analyze")
async def analyze(
    file: UploadFile = File(...),
    task: str = Form(
        default="<MORE_DETAILED_CAPTION>",
        description=(
            "Florence-2 task token. Options: "
            "<CAPTION>, <DETAILED_CAPTION>, <MORE_DETAILED_CAPTION>, "
            "<OCR>, <OBJECT_DETECTION>, <DENSE_REGION_CAPTION>, "
            "<REGION_PROPOSAL>, <CAPTION_TO_PHRASE_GROUNDING>"
        ),
    ),
    text_input: str | None = Form(default=None),
):
    """Generic endpoint — caller chooses the task token."""
    _check_ready()
    img = _load_image(await file.read())
    result = _infer(img, task, text_input)
    return JSONResponse({"task": task, "result": result})


@app.post("/caption")
async def caption(file: UploadFile = File(...)):
    """One-sentence caption."""
    _check_ready()
    img = _load_image(await file.read())
    result = _infer(img, "<CAPTION>")
    return JSONResponse({"caption": result.get("<CAPTION>", "")})


@app.post("/detailed-caption")
async def detailed_caption(file: UploadFile = File(...)):
    """Paragraph-level detailed caption."""
    _check_ready()
    img = _load_image(await file.read())
    result = _infer(img, "<MORE_DETAILED_CAPTION>")
    return JSONResponse({"caption": result.get("<MORE_DETAILED_CAPTION>", "")})


@app.post("/ocr")
async def ocr(file: UploadFile = File(...)):
    """Extract all text visible in the image."""
    _check_ready()
    img = _load_image(await file.read())
    result = _infer(img, "<OCR>")
    return JSONResponse({"text": result.get("<OCR>", "")})


@app.post("/object-detection")
async def object_detection(file: UploadFile = File(...)):
    """
    Detect objects. Returns list of {label, bbox:[x1,y1,x2,y2]}.
    """
    _check_ready()
    img = _load_image(await file.read())
    result = _infer(img, "<OBJECT_DETECTION>")
    raw = result.get("<OBJECT_DETECTION>", {})

    labels = raw.get("labels", [])
    bboxes = raw.get("bboxes", [])
    objects = [
        {"label": lbl, "bbox": list(map(int, box))}
        for lbl, box in zip(labels, bboxes)
    ]
    return JSONResponse({"objects": objects, "count": len(objects)})
