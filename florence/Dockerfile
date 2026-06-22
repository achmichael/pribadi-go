# ─────────────────────────────────────────────────────────────────────────────
# Stage: runtime
# Base: official PyTorch image with CUDA 12.1 + cuDNN 8 on Ubuntu 22.04
# Florence-2-large-ft requires torch >= 2.0, transformers >= 4.38
# ─────────────────────────────────────────────────────────────────────────────
FROM pytorch/pytorch:2.3.1-cuda12.1-cudnn8-runtime

# Prevent interactive prompts during apt
ENV DEBIAN_FRONTEND=noninteractive

# System deps: git (for huggingface hub clone fallback), libgl for Pillow
RUN apt-get update && apt-get install -y --no-install-recommends \
        git \
        libgl1 \
        libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# ── Python dependencies ───────────────────────────────────────────────────────
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# ── Application code ──────────────────────────────────────────────────────────
COPY main.py .

# Model cache directory — mount a host volume here to persist across restarts
# so the model is not re-downloaded every container start
ENV HF_HOME=/model_cache

# Expose FastAPI port
EXPOSE 8000

# Pre-download model weights at build time so first request is not slow.
# Comment this out if you prefer lazy loading (saves build time / image size).
# RUN python - <<'EOF'
# from transformers import AutoProcessor, AutoModelForCausalLM
# model_id = "microsoft/Florence-2-large-ft"
# AutoProcessor.from_pretrained(model_id, trust_remote_code=True)
# AutoModelForCausalLM.from_pretrained(model_id, trust_remote_code=True)
# print("Florence-2 weights downloaded successfully.")
# EOF

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
