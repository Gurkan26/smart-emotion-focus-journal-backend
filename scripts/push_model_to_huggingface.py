#!/usr/bin/env python3
"""
Uploads the trained PEFT (LoRA) adapter to Hugging Face Model Hub.
MasterFabric Academy - AI Model Deployment Pipeline
"""

import os
import sys
import argparse

try:
    from huggingface_hub import HfApi, login
except ImportError:
    print("[!] Error: huggingface_hub is missing. Install with 'pip install huggingface_hub'")
    sys.exit(1)

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_ADAPTER_DIR = os.path.abspath(os.path.join(SCRIPT_DIR, "..", "peft-adapters", "gemma-journal-custom-lora"))
DEFAULT_REPO_ID = "Gurkan26/gemma-journal-custom-lora"


def push_model(adapter_dir: str, repo_id: str, token: str = None):
    if not os.path.exists(adapter_dir):
        print(f"[!] Error: Adapter directory not found: {adapter_dir}")
        sys.exit(1)

    hf_token = token or os.getenv("HF_TOKEN")
    if not hf_token:
        print("[!] Error: HF_TOKEN is required. Pass --token or set HF_TOKEN env var.")
        sys.exit(1)

    try:
        print(f"[*] Authenticating with Hugging Face...")
        login(token=hf_token)

        api = HfApi()
        print(f"[*] Ensuring model repository exists: {repo_id}")
        api.create_repo(repo_id=repo_id, repo_type="model", exist_ok=True)

        print(f"[*] Uploading LoRA adapter folder from '{adapter_dir}' to '{repo_id}'...")
        api.upload_folder(
            folder_path=adapter_dir,
            repo_id=repo_id,
            repo_type="model",
            commit_message="MasterFabric Pipeline: Upload trained PEFT LoRA adapter weights"
        )
        print(f"[+] Successfully pushed model adapter to https://huggingface.co/{repo_id}")

    except Exception as e:
        print(f"[!] Failed to push model adapter: {e}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="Push PEFT LoRA adapter to Hugging Face Model Hub.")
    parser.add_argument("--adapter-dir", default=DEFAULT_ADAPTER_DIR, help="Path to LoRA adapter folder")
    parser.add_argument("--repo-id", default=DEFAULT_REPO_ID, help="Hugging Face Model Hub repository ID")
    parser.add_argument("--token", help="Hugging Face API token")

    args = parser.parse_args()
    push_model(args.adapter_dir, args.repo_id, args.token)


if __name__ == "__main__":
    main()
