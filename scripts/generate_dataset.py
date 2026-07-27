#!/usr/bin/env python3
"""
Dataset Generator for Emotion & Focus Journal PEFT Fine-Tuning.
Generates synthetic/structured JSONL training pairs for prompt optimization and cognitive state evaluation.
"""

import json
import os

DATASET_FILE = os.path.join(os.path.dirname(__file__), "journal_finetune_dataset.jsonl")

SAMPLE_DATA = [
    {
        "instruction": "Optimize the following user journal entry for emotional sentiment extraction and focus rating.",
        "input": "Bugün işte biraz stresliydim ama 2 saat kesintisiz kod yazdım ve bir modülü bitirdim.",
        "output": "{\"sentiment\": \"mildly_stressed_accomplished\", \"focus_score\": 85, \"key_themes\": [\"work_stress\", \"deep_work\", \"completion\"], \"optimized_prompt\": \"Analyze entry for dual-state emotion (stress vs achievement) and deep work duration (120 mins).\"}"
    },
    {
        "instruction": "Optimize the following user journal entry for emotional sentiment extraction and focus rating.",
        "input": "Tüm gün toplantılar yüzünden hiçbir şey yapamadım, sürekli dikkatim dağıldı.",
        "output": "{\"sentiment\": \"frustrated_overwhelmed\", \"focus_score\": 30, \"key_themes\": [\"meeting_fatigue\", \"distraction\", \"low_productivity\"], \"optimized_prompt\": \"Extract cognitive load metrics emphasizing meeting overload and context switching penalty.\"}"
    },
    {
        "instruction": "Optimize prompt for cognitive recovery and mindfulness advice.",
        "input": "Akşam yürüyüşe çıktım, kafam toplandı.",
        "output": "{\"sentiment\": \"refreshed_calm\", \"focus_score\": 75, \"key_themes\": [\"active_recovery\", \"mental_clarity\"], \"optimized_prompt\": \"Evaluate restoration effect of physical activity on cognitive capacity.\"}"
    }
]

def generate():
    os.makedirs(os.path.dirname(DATASET_FILE), exist_ok=True)
    with open(DATASET_FILE, "w", encoding="utf-8") as f:
        for item in SAMPLE_DATA:
            f.write(json.dumps(item, ensure_ascii=False) + "\n")
    print(f"[+] Dataset successfully generated at {DATASET_FILE} ({len(SAMPLE_DATA)} samples)")

if __name__ == "__main__":
    generate()
