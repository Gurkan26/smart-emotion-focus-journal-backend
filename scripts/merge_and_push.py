#!/usr/bin/env python3
"""
Merges the trained PEFT (LoRA) adapter into the base Gemma model
and uploads the full merged model to Hugging Face Model Hub.
MasterFabric Academy - AI Model Merge & Deployment Pipeline
"""

import os
import sys
import argparse
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel
from huggingface_hub import HfApi, login

def main():
    parser = argparse.ArgumentParser(description="Merge LoRA adapter into base model and push to HF")
    parser.add_argument("--base-model", default="google/gemma-2b-it", help="Base model identifier")
    parser.add_argument("--adapter-dir", default="./peft-adapters/gemma-journal-custom-lora", help="Path to LoRA adapter folder")
    parser.add_argument("--output-dir", default="./gemma-journal-merged", help="Directory to save merged model")
    parser.add_argument("--repo-id", default="Gurkan26/gemma-journal-merged", help="Hugging Face repo ID for merged model")
    parser.add_argument("--token", help="Hugging Face API token")
    args = parser.parse_args()

    token = args.token or os.getenv("HF_TOKEN")
    if not token:
        print("[!] Error: Hugging Face Token is required. Pass --token or set HF_TOKEN env var.")
        sys.exit(1)

    is_hf_repo = "/" in args.adapter_dir and not os.path.exists(args.adapter_dir)
    if not is_hf_repo and not os.path.exists(args.adapter_dir):
        print(f"[!] Error: Adapter directory not found: {args.adapter_dir}")
        sys.exit(1)

    try:
        print(f"[*] Loading base model '{args.base_model}'...")
        # Load in float16 for resource optimization
        base_model = AutoModelForCausalLM.from_pretrained(
            args.base_model,
            torch_dtype=torch.float16,
            device_map="auto",
            trust_remote_code=True,
            token=token
        )
        tokenizer = AutoTokenizer.from_pretrained(args.base_model, token=token)

        print(f"[*] Loading LoRA adapter from '{args.adapter_dir}'...")
        model = PeftModel.from_pretrained(base_model, args.adapter_dir, token=token)

        print("[*] Merging LoRA weights into base model...")
        merged_model = model.merge_and_unload()

        print(f"[*] Saving merged model locally to '{args.output_dir}'...")
        merged_model.save_pretrained(args.output_dir)
        tokenizer.save_pretrained(args.output_dir)
        print("[+] Merged model saved successfully.")

        # Free memory before upload
        del base_model
        del model
        del merged_model
        if torch.cuda.is_available():
            torch.cuda.empty_cache()

        print(f"\n[*] Authenticating with Hugging Face...")
        login(token=token)

        api = HfApi()
        print(f"[*] Ensuring repository exists: {args.repo_id}")
        api.create_repo(repo_id=args.repo_id, repo_type="model", exist_ok=True)

        print(f"[*] Uploading full merged model from '{args.output_dir}' to '{args.repo_id}' (this might take a few minutes)...")
        api.upload_folder(
            folder_path=args.output_dir,
            repo_id=args.repo_id,
            repo_type="model",
            commit_message="MasterFabric Pipeline: Upload merged fine-tuned Gemma model"
        )
        print(f"\n[+] SUCCESS! Successfully pushed merged model to: https://huggingface.co/{args.repo_id}")

    except Exception as e:
        print(f"[!] Failed during merge and push pipeline: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
