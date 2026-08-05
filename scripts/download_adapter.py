#!/usr/bin/env python3
"""
Downloads the trained LoRA adapter from Hugging Face Hub to the local machine.
"""

import os
from huggingface_hub import snapshot_download

def main():
    repo_id = "Gurkan26/gemma-journal-custom-lora"
    token = os.getenv("HF_TOKEN")
    if not token:
        print("[!] Warning: HF_TOKEN environment variable not set. Attempting public download...")

    print(f"[*] Downloading adapter weights from HF repository '{repo_id}'...")
    try:
        snapshot_download(
            repo_id=repo_id,
            local_dir=local_dir,
            token=token
        )
        print(f"[+] Success! Files downloaded to: {local_dir}")
    except Exception as e:
        print(f"[!] Error downloading: {e}")

if __name__ == "__main__":
    main()
