#!/usr/bin/env python3
"""
PEFT (LoRA/QLoRA) Fine-Tuning Script with Live Telemetry Reporting.
Trains a real LoRA adapter on journal emotion & focus prompt dataset using PyTorch
and HuggingFace Transformers + PEFT, reporting live loss/epoch metrics
to the Go Backend for real-time monitoring in the Admin Cockpit UI.

Supports:
  - GPU (CUDA) training when available
  - CPU fallback for systems without sufficient VRAM
  - Gradient accumulation for low-memory GPUs (e.g. GTX 1050 Ti 4GB)
  - Real LoRA adapter weight saving in safetensors format
"""

import sys
import os
import time
import argparse
import json
import math

try:
    import requests
except ImportError:
    requests = None

try:
    import psutil
except ImportError:
    psutil = None

import torch
from torch.utils.data import Dataset, DataLoader
from transformers import AutoTokenizer, AutoModelForCausalLM, get_linear_schedule_with_warmup
from peft import LoraConfig, get_peft_model, TaskType

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DATASET = os.path.join(SCRIPT_DIR, "journal_finetune_dataset.jsonl")
DEFAULT_OUTPUT_DIR = os.path.join(SCRIPT_DIR, "..", "peft-adapters")


# ---------------------------------------------------------------------------
# Dataset
# ---------------------------------------------------------------------------
class JournalDataset(Dataset):
    """Loads JSONL instruction/input/output triples and tokenizes them for causal LM training."""

    def __init__(self, jsonl_path, tokenizer, max_length=512):
        self.samples = []
        self.tokenizer = tokenizer
        self.max_length = max_length

        if not os.path.isfile(jsonl_path):
            raise FileNotFoundError(f"Dataset not found: {jsonl_path}. Run generate_dataset.py first.")

        with open(jsonl_path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    item = json.loads(line)
                    self.samples.append(item)
                except json.JSONDecodeError:
                    continue

        if len(self.samples) == 0:
            raise ValueError(f"No valid samples found in {jsonl_path}")

        print(f"[+] Loaded {len(self.samples)} training samples from {jsonl_path}")

    def __len__(self):
        return len(self.samples)

    def __getitem__(self, idx):
        item = self.samples[idx]

        # Format as instruction-following prompt
        text = (
            f"### Instruction:\n{item['instruction']}\n\n"
            f"### Input:\n{item['input']}\n\n"
            f"### Response:\n{item['output']}"
        )

        encoding = self.tokenizer(
            text,
            truncation=True,
            max_length=self.max_length,
            padding="max_length",
            return_tensors="pt"
        )

        input_ids = encoding["input_ids"].squeeze(0)
        attention_mask = encoding["attention_mask"].squeeze(0)

        return {
            "input_ids": input_ids,
            "attention_mask": attention_mask,
            "labels": input_ids.clone()  # Causal LM: labels = input_ids
        }


# ---------------------------------------------------------------------------
# Telemetry
# ---------------------------------------------------------------------------
def send_telemetry(backend_url, payload):
    """Send training telemetry to the Go backend for live Admin Cockpit monitoring."""
    if not backend_url:
        return False
    endpoint = f"{backend_url.rstrip('/')}/api/admin/finetune/telemetry"
    if requests is not None:
        try:
            resp = requests.post(endpoint, json=payload, timeout=3)
            return resp.status_code == 200
        except Exception as e:
            print(f"[!] Telemetry send failed: {e}", file=sys.stderr)
    return False


def get_memory_usage(device):
    """Get current memory usage (VRAM for CUDA, RAM for CPU)."""
    if device.type == "cuda":
        allocated = torch.cuda.memory_allocated(device) / (1024 ** 3)
        return round(allocated, 2)
    elif psutil is not None:
        process = psutil.Process(os.getpid())
        mem_gb = process.memory_info().rss / (1024 ** 3)
        return round(mem_gb, 2)
    return 0.0


# ---------------------------------------------------------------------------
# Args
# ---------------------------------------------------------------------------
def parse_args():
    parser = argparse.ArgumentParser(description="PEFT Fine-Tuning Engine for Gemma/Llama/Qwen Models")
    parser.add_argument("--model", type=str, default="google/gemma-2b-it", help="HuggingFace base model identifier")
    parser.add_argument("--dataset", type=str, default=DEFAULT_DATASET, help="Path to JSONL training dataset")
    parser.add_argument("--lora-r", type=int, default=8, help="LoRA rank dimension")
    parser.add_argument("--lora-alpha", type=int, default=16, help="LoRA alpha scaling factor")
    parser.add_argument("--lora-dropout", type=float, default=0.05, help="LoRA dropout rate")
    parser.add_argument("--epochs", type=int, default=3, help="Number of training epochs")
    parser.add_argument("--batch-size", type=int, default=1, help="Training batch size (keep low for 4GB VRAM)")
    parser.add_argument("--grad-accum", type=int, default=4, help="Gradient accumulation steps")
    parser.add_argument("--learning-rate", type=float, default=2e-4, help="Initial learning rate")
    parser.add_argument("--max-length", type=int, default=256, help="Max token sequence length")
    parser.add_argument("--backend-url", type=str, default="http://localhost:8080", help="Go Backend URL for telemetry")
    parser.add_argument("--adapter-name", type=str, default="gemma-journal-custom-lora", help="Output PEFT adapter name")
    parser.add_argument("--output-dir", type=str, default=DEFAULT_OUTPUT_DIR, help="Directory to save adapter weights")
    parser.add_argument("--force-cpu", action="store_true", help="Force CPU training even if CUDA is available")
    parser.add_argument("--token", type=str, default=os.getenv("HF_TOKEN"), help="HuggingFace API token for gated models")
    return parser.parse_args()


# ---------------------------------------------------------------------------
# Main Training Loop
# ---------------------------------------------------------------------------
def main():
    args = parse_args()

    print("=" * 60)
    print("   MASTERFABRIC PEFT (LoRA) REAL FINE-TUNING ENGINE")
    print("=" * 60)

    # ---- Device Selection ----
    if args.force_cpu:
        device = torch.device("cpu")
        print("[*] Device           : CPU (forced)")
    elif torch.cuda.is_available():
        device = torch.device("cuda")
        gpu_name = torch.cuda.get_device_name(0)
        vram_total = torch.cuda.get_device_properties(0).total_memory / (1024 ** 3)
        print(f"[*] Device           : CUDA ({gpu_name}, {vram_total:.1f} GB VRAM)")
    else:
        device = torch.device("cpu")
        print("[*] Device           : CPU (CUDA not available)")

    print(f"[*] Base Model       : {args.model}")
    print(f"[*] LoRA Config      : r={args.lora_r}, alpha={args.lora_alpha}, dropout={args.lora_dropout}")
    print(f"[*] Epochs           : {args.epochs}")
    print(f"[*] Batch Size       : {args.batch_size} (grad accum: {args.grad_accum})")
    print(f"[*] Max Seq Length   : {args.max_length}")
    print(f"[*] Learning Rate    : {args.learning_rate}")
    print(f"[*] Dataset          : {args.dataset}")
    print(f"[*] Output Adapter   : {args.adapter_name}")
    print(f"[*] Backend Telemetry: {args.backend_url}")
    print("=" * 60)

    # ---- Send initial telemetry ----
    start_time = time.time()
    send_telemetry(args.backend_url, {
        "status": "RUNNING",
        "current_epoch": 0,
        "total_epochs": args.epochs,
        "loss": 0.0,
        "learning_rate": args.learning_rate,
        "vram_gb": 0.0,
        "adapter_name": args.adapter_name,
        "message": f"Initializing model '{args.model}' and PEFT LoRA configuration..."
    })

    token = args.token or os.getenv("HF_TOKEN")

    # ---- Load Tokenizer ----
    print("\n[1/5] Loading tokenizer...")
    tokenizer = AutoTokenizer.from_pretrained(args.model, trust_remote_code=True, token=token)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
    print(f"      Tokenizer loaded. Vocab size: {tokenizer.vocab_size}")

    # ---- Load Base Model ----
    print("\n[2/5] Loading base model...")
    send_telemetry(args.backend_url, {
        "status": "RUNNING",
        "current_epoch": 0,
        "total_epochs": args.epochs,
        "loss": 0.0,
        "learning_rate": args.learning_rate,
        "vram_gb": get_memory_usage(device),
        "adapter_name": args.adapter_name,
        "message": "Loading base model weights..."
    })

    model_kwargs = {
        "trust_remote_code": True,
        "token": token,
    }

    # For CUDA devices with limited VRAM, use float16
    if device.type == "cuda":
        model_kwargs["dtype"] = torch.float16
    else:
        model_kwargs["dtype"] = torch.float32

    model = AutoModelForCausalLM.from_pretrained(args.model, **model_kwargs)
    model.config.use_cache = False  # Required for gradient checkpointing compatibility

    total_params = sum(p.numel() for p in model.parameters())
    print(f"      Base model loaded. Total params: {total_params:,}")

    # ---- Apply LoRA ----
    print("\n[3/5] Applying PEFT LoRA configuration...")

    # Detect target modules based on model architecture
    target_modules = _detect_target_modules(model)
    print(f"      Target modules: {target_modules}")

    lora_config = LoraConfig(
        task_type=TaskType.CAUSAL_LM,
        r=args.lora_r,
        lora_alpha=args.lora_alpha,
        lora_dropout=args.lora_dropout,
        target_modules=target_modules,
        bias="none",
    )

    model = get_peft_model(model, lora_config)
    model.to(device)

    trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)
    total_params_after = sum(p.numel() for p in model.parameters())
    pct = 100.0 * trainable_params / total_params_after
    print(f"      LoRA applied. Trainable: {trainable_params:,} / {total_params_after:,} ({pct:.2f}%)")

    # Enable input require grads for autograd backward pass
    if hasattr(model, "enable_input_require_grads"):
        model.enable_input_require_grads()
    elif hasattr(model, "get_input_embeddings"):
        def make_inputs_require_grad(module, input, output):
            output.requires_grad_(True)
        model.get_input_embeddings().register_forward_hook(make_inputs_require_grad)

    # Enable gradient checkpointing for memory savings on GPU
    if device.type == "cuda":
        try:
            model.gradient_checkpointing_enable(gradient_checkpointing_kwargs={"use_reentrant": False})
            print("      Gradient checkpointing enabled for VRAM savings")
        except Exception:
            try:
                model.gradient_checkpointing_enable()
                print("      Gradient checkpointing enabled for VRAM savings")
            except Exception:
                pass

    # ---- Load Dataset ----
    print("\n[4/5] Loading training dataset...")
    dataset = JournalDataset(args.dataset, tokenizer, max_length=args.max_length)
    dataloader = DataLoader(dataset, batch_size=args.batch_size, shuffle=True, drop_last=False)

    total_steps_per_epoch = math.ceil(len(dataloader) / args.grad_accum)
    total_steps = total_steps_per_epoch * args.epochs
    print(f"      DataLoader: {len(dataloader)} batches/epoch, ~{total_steps} optimizer steps total")

    # ---- Optimizer & Scheduler ----
    optimizer = torch.optim.AdamW(
        filter(lambda p: p.requires_grad, model.parameters()),
        lr=args.learning_rate,
        weight_decay=0.01
    )
    scheduler = get_linear_schedule_with_warmup(
        optimizer,
        num_warmup_steps=max(1, total_steps // 10),
        num_training_steps=total_steps
    )

    # ---- Training Loop ----
    print(f"\n[5/5] Starting training ({args.epochs} epochs)...\n")
    model.train()
    global_step = 0
    best_loss = float("inf")
    loss_history = []

    for epoch in range(1, args.epochs + 1):
        epoch_loss = 0.0
        epoch_steps = 0
        optimizer.zero_grad()

        for batch_idx, batch in enumerate(dataloader, 1):
            input_ids = batch["input_ids"].to(device)
            attention_mask = batch["attention_mask"].to(device)
            labels = batch["labels"].to(device)

            outputs = model(
                input_ids=input_ids,
                attention_mask=attention_mask,
                labels=labels,
            )
            loss = outputs.loss / args.grad_accum
            loss.backward()

            current_loss = outputs.loss.item()
            epoch_loss += current_loss
            epoch_steps += 1

            # Gradient accumulation step
            if batch_idx % args.grad_accum == 0 or batch_idx == len(dataloader):
                torch.nn.utils.clip_grad_norm_(model.parameters(), max_norm=1.0)
                optimizer.step()
                scheduler.step()
                optimizer.zero_grad()
                global_step += 1

                current_lr = scheduler.get_last_lr()[0]
                mem_used = get_memory_usage(device)
                progress_pct = (global_step / total_steps) * 100.0

                log_msg = (
                    f"[Epoch {epoch}/{args.epochs}] "
                    f"Step {global_step}/{total_steps} - "
                    f"Loss: {current_loss:.4f} - "
                    f"LR: {current_lr:.2e} - "
                    f"Mem: {mem_used:.2f} GB"
                )
                print(log_msg)

                send_telemetry(args.backend_url, {
                    "status": "RUNNING",
                    "current_epoch": epoch,
                    "total_epochs": args.epochs,
                    "step": global_step,
                    "total_steps": total_steps,
                    "progress_pct": round(progress_pct, 1),
                    "loss": round(current_loss, 4),
                    "learning_rate": current_lr,
                    "vram_gb": mem_used,
                    "adapter_name": args.adapter_name,
                    "message": log_msg
                })

        avg_epoch_loss = epoch_loss / max(epoch_steps, 1)
        loss_history.append(avg_epoch_loss)
        print(f"\n--- Epoch {epoch} complete. Avg Loss: {avg_epoch_loss:.4f} ---\n")

        if avg_epoch_loss < best_loss:
            best_loss = avg_epoch_loss

    # ---- Save LoRA Adapter ----
    elapsed = round(time.time() - start_time, 2)
    adapter_dir = os.path.join(args.output_dir, args.adapter_name)
    os.makedirs(adapter_dir, exist_ok=True)

    print(f"\n[+] Saving LoRA adapter to {adapter_dir}...")
    model.save_pretrained(adapter_dir)
    tokenizer.save_pretrained(adapter_dir)

    # Verify saved files
    saved_files = os.listdir(adapter_dir)
    print(f"[+] Saved files: {saved_files}")

    print(f"\n{'=' * 60}")
    print(f"  FINE-TUNING COMPLETED SUCCESSFULLY")
    print(f"{'=' * 60}")
    print(f"[+] Final Avg Loss  : {best_loss:.4f}")
    print(f"[+] Total Time      : {elapsed}s")
    print(f"[+] Adapter Dir     : {adapter_dir}")
    print(f"[+] Loss History    : {[f'{l:.4f}' for l in loss_history]}")
    print(f"{'=' * 60}")

    send_telemetry(args.backend_url, {
        "status": "COMPLETED",
        "current_epoch": args.epochs,
        "total_epochs": args.epochs,
        "progress_pct": 100.0,
        "loss": round(best_loss, 4),
        "learning_rate": 0.0,
        "vram_gb": 0.0,
        "adapter_name": args.adapter_name,
        "message": f"Training completed in {elapsed}s. Final loss: {best_loss:.4f}. Adapter '{args.adapter_name}' saved and ready for hot-swap."
    })


def _detect_target_modules(model):
    """Auto-detect LoRA target linear layers based on model architecture."""
    # Common target module names across architectures
    candidates = ["q_proj", "v_proj", "k_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]
    found = set()

    for name, module in model.named_modules():
        for candidate in candidates:
            if candidate in name and isinstance(module, torch.nn.Linear):
                found.add(candidate)

    if found:
        # For memory-constrained GPUs, limit to q_proj and v_proj
        if torch.cuda.is_available():
            vram_gb = torch.cuda.get_device_properties(0).total_memory / (1024 ** 3)
            if vram_gb < 6.0 and len(found) > 2:
                limited = {"q_proj", "v_proj"} & found
                if limited:
                    return list(limited)
        return list(found)

    # Fallback: scan for any Linear layers
    linear_names = set()
    for name, module in model.named_modules():
        if isinstance(module, torch.nn.Linear):
            parts = name.split(".")
            linear_names.add(parts[-1])

    if linear_names:
        return list(linear_names)[:4]

    return ["q_proj", "v_proj"]


if __name__ == "__main__":
    main()
