import os
import transformers.dynamic_module_utils

_original_get_imports = transformers.dynamic_module_utils.get_imports

def _patched_get_imports(filename: str | os.PathLike) -> list[str]:
    imports = _original_get_imports(filename)
    if "flash_attn" in imports:
        imports.remove("flash_attn")
    return imports

transformers.dynamic_module_utils.get_imports = _patched_get_imports

from transformers import AutoModelForCausalLM
print("Loading model...")
model = AutoModelForCausalLM.from_pretrained(
    "microsoft/Florence-2-large-ft",
    trust_remote_code=True,
    attn_implementation="eager"
)
print("Model loaded!")
