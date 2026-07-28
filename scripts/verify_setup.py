import torch
print("PyTorch version:", torch.__version__)
print("CUDA available:", torch.cuda.is_available())
if torch.cuda.is_available():
    print("GPU:", torch.cuda.get_device_name(0))
    print("VRAM:", round(torch.cuda.get_device_properties(0).total_mem / (1024**3), 1), "GB")
from transformers import AutoModelForCausalLM
print("transformers AutoModelForCausalLM: OK")
from peft import LoraConfig, get_peft_model
print("peft LoraConfig: OK")
print("ALL CHECKS PASSED")
