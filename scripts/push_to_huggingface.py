#!/usr/bin/env python3
"""
Uploads prompt optimization dataset to Hugging Face Datasets Hub in Parquet format.
MasterFabric Academy - AI Dataset Deployment Pipeline
"""

import os
import sys
import argparse

try:
    import pandas as pd
    from datasets import Dataset
except ImportError:
    print("[!] Error: Missing dependencies. Install with 'pip install pandas datasets'")
    sys.exit(1)

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_CSV_PATH = os.path.abspath(os.path.join(SCRIPT_DIR, "..", "prompt_dataset.csv"))
DEFAULT_REPO_ID = "Gurkan26/enterprise-prompt-optimizer"


def push_dataset(csv_path: str, repo_id: str, token: str = None):
    if not os.path.exists(csv_path):
        print(f"[!] Error: CSV dataset file not found: {csv_path}")
        sys.exit(1)

    hf_token = token or os.getenv("HF_TOKEN")
    if not hf_token:
        print("[!] Error: HF_TOKEN is required. Pass --token or set HF_TOKEN env var.")
        sys.exit(1)

    try:
        print(f"[*] Loading dataset from CSV: {csv_path}")
        df = pd.read_csv(csv_path)
        print(f"[*] Dataset loaded successfully ({len(df)} rows, columns: {list(df.columns)})")

        print("[*] Converting pandas DataFrame to Hugging Face Dataset...")
        hf_dataset = Dataset.from_pandas(df)

        print(f"[*] Pushing dataset to Hugging Face Hub: {repo_id}...")
        hf_dataset.push_to_hub(
            repo_id=repo_id,
            private=False,
            token=hf_token
        )
        print(f"[+] Successfully pushed dataset to https://huggingface.co/datasets/{repo_id}")

    except Exception as e:
        print(f"[!] Failed to push dataset to Hugging Face: {e}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="Push prompt dataset to Hugging Face Datasets Hub.")
    parser.add_argument("--csv", default=DEFAULT_CSV_PATH, help="Path to prompt dataset CSV file")
    parser.add_argument("--repo-id", default=DEFAULT_REPO_ID, help="Hugging Face Dataset Hub repository ID")
    parser.add_argument("--token", help="Hugging Face API token")

    args = parser.parse_args()
    push_dataset(args.csv, args.repo_id, args.token)


if __name__ == "__main__":
    main()