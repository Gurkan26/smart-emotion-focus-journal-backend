#!/usr/bin/env python3
"""
Dataset Generator for Emotion & Focus Journal PEFT Fine-Tuning.
Extracts real journal entries and prompt optimizations from PostgreSQL,
then generates structured JSONL training pairs for LoRA fine-tuning.
Falls back to built-in sample data when the database is unavailable.
"""

import json
import os
import sys
import argparse

# Optional PostgreSQL driver
try:
    import psycopg2
except ImportError:
    psycopg2 = None

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DATASET_FILE = os.path.join(SCRIPT_DIR, "journal_finetune_dataset.jsonl")

# Built-in fallback samples used when no database is available
FALLBACK_SAMPLES = [
    {
        "instruction": "Optimize the following user journal entry for emotional sentiment extraction and focus rating.",
        "input": "Bugün işte biraz stresliydim ama 2 saat kesintisiz kod yazdım ve bir modülü bitirdim.",
        "output": json.dumps({
            "sentiment": "mildly_stressed_accomplished",
            "focus_score": 85,
            "key_themes": ["work_stress", "deep_work", "completion"],
            "optimized_prompt": "Analyze entry for dual-state emotion (stress vs achievement) and deep work duration (120 mins)."
        }, ensure_ascii=False)
    },
    {
        "instruction": "Optimize the following user journal entry for emotional sentiment extraction and focus rating.",
        "input": "Tüm gün toplantılar yüzünden hiçbir şey yapamadım, sürekli dikkatim dağıldı.",
        "output": json.dumps({
            "sentiment": "frustrated_overwhelmed",
            "focus_score": 30,
            "key_themes": ["meeting_fatigue", "distraction", "low_productivity"],
            "optimized_prompt": "Extract cognitive load metrics emphasizing meeting overload and context switching penalty."
        }, ensure_ascii=False)
    },
    {
        "instruction": "Optimize prompt for cognitive recovery and mindfulness advice.",
        "input": "Akşam yürüyüşe çıktım, kafam toplandı.",
        "output": json.dumps({
            "sentiment": "refreshed_calm",
            "focus_score": 75,
            "key_themes": ["active_recovery", "mental_clarity"],
            "optimized_prompt": "Evaluate restoration effect of physical activity on cognitive capacity."
        }, ensure_ascii=False)
    },
    {
        "instruction": "Analyze the user's emotional state and cognitive focus from their journal entry.",
        "input": "Sabah erken kalktım, meditasyon yaptım. Gün boyunca çok verimli çalıştım.",
        "output": json.dumps({
            "sentiment": "energized_focused",
            "focus_score": 92,
            "key_themes": ["morning_routine", "meditation", "high_productivity"],
            "optimized_prompt": "Evaluate positive morning routine impact on sustained cognitive performance."
        }, ensure_ascii=False)
    },
    {
        "instruction": "Analyze the user's emotional state and cognitive focus from their journal entry.",
        "input": "Uyuyamadım, kafamda sürekli yarınki sunum dönüp durdu. Çok gerginim.",
        "output": json.dumps({
            "sentiment": "anxious_sleep_deprived",
            "focus_score": 25,
            "key_themes": ["insomnia", "anticipatory_anxiety", "rumination"],
            "optimized_prompt": "Detect pre-performance anxiety indicators and sleep quality degradation."
        }, ensure_ascii=False)
    },
    {
        "instruction": "Optimize the following user journal entry for emotional sentiment extraction and focus rating.",
        "input": "Bugün yeni bir proje başlattım, çok heyecanlıyım ama biraz da endişeliyim.",
        "output": json.dumps({
            "sentiment": "excited_apprehensive",
            "focus_score": 70,
            "key_themes": ["new_project", "mixed_emotions", "motivation"],
            "optimized_prompt": "Analyze mixed emotional state balancing excitement with uncertainty in new endeavors."
        }, ensure_ascii=False)
    },
    {
        "instruction": "Analyze the user's emotional state and cognitive focus from their journal entry.",
        "input": "Arkadaşlarımla buluştum, güzel vakit geçirdik. Moralim çok düzeldi.",
        "output": json.dumps({
            "sentiment": "happy_socially_fulfilled",
            "focus_score": 65,
            "key_themes": ["social_connection", "mood_improvement", "relaxation"],
            "optimized_prompt": "Evaluate social interaction impact on emotional well-being and mood restoration."
        }, ensure_ascii=False)
    },
    {
        "instruction": "Optimize prompt for cognitive recovery and mindfulness advice.",
        "input": "Tüm gün ekran başında kaldım, gözlerim ve başım çok ağrıyor.",
        "output": json.dumps({
            "sentiment": "fatigued_strained",
            "focus_score": 35,
            "key_themes": ["screen_fatigue", "physical_strain", "burnout_risk"],
            "optimized_prompt": "Detect digital eye strain and prolonged screen exposure effects on cognitive capacity."
        }, ensure_ascii=False)
    },
]


def parse_args():
    parser = argparse.ArgumentParser(description="Generate JSONL dataset for PEFT fine-tuning")
    parser.add_argument("--db-host", type=str, default="localhost", help="PostgreSQL host")
    parser.add_argument("--db-port", type=int, default=5433, help="PostgreSQL port")
    parser.add_argument("--db-name", type=str, default="masterfabric", help="Database name")
    parser.add_argument("--db-user", type=str, default="masterfabric", help="Database user")
    parser.add_argument("--db-password", type=str, default="masterfabric", help="Database password")
    parser.add_argument("--output", type=str, default=DATASET_FILE, help="Output JSONL file path")
    parser.add_argument("--skip-db", action="store_true", help="Skip database and use only fallback samples")
    return parser.parse_args()


def fetch_journals_from_db(args):
    """Fetch real journal entries from PostgreSQL journals table."""
    if psycopg2 is None:
        print("[!] psycopg2 not installed. Run: pip install psycopg2-binary")
        return []

    try:
        conn = psycopg2.connect(
            host=args.db_host,
            port=args.db_port,
            dbname=args.db_name,
            user=args.db_user,
            password=args.db_password,
            connect_timeout=5
        )
        cursor = conn.cursor()

        samples = []

        # 1. Fetch journals with decision scores
        cursor.execute("""
            SELECT content, decision_score
            FROM journals
            WHERE content IS NOT NULL AND LENGTH(content) > 10
            ORDER BY created_at DESC
            LIMIT 500
        """)
        rows = cursor.fetchall()
        print(f"[+] Fetched {len(rows)} journal entries from database")

        for content, score in rows:
            # Derive sentiment category from decision score
            if score >= 80:
                sentiment = "focused_productive"
                themes = ["high_focus", "deep_work", "achievement"]
            elif score >= 60:
                sentiment = "moderately_focused"
                themes = ["moderate_focus", "steady_progress"]
            elif score >= 40:
                sentiment = "distracted_mixed"
                themes = ["mixed_attention", "context_switching"]
            elif score >= 20:
                sentiment = "stressed_unfocused"
                themes = ["stress", "low_focus", "overwhelmed"]
            else:
                sentiment = "fatigued_burnt_out"
                themes = ["burnout", "exhaustion", "disengagement"]

            output_obj = {
                "sentiment": sentiment,
                "focus_score": int(score),
                "key_themes": themes,
                "optimized_prompt": f"Analyze journal entry with decision score {score:.0f} for cognitive state indicators."
            }
            samples.append({
                "instruction": "Analyze the user's emotional state and cognitive focus from their journal entry.",
                "input": content.strip(),
                "output": json.dumps(output_obj, ensure_ascii=False)
            })

        # 2. Fetch prompt optimizations if available
        cursor.execute("""
            SELECT original_prompt, optimized_prompt, template
            FROM prompt_optimizations
            WHERE original_prompt IS NOT NULL AND LENGTH(original_prompt) > 10
            ORDER BY created_at DESC
            LIMIT 200
        """)
        opt_rows = cursor.fetchall()
        print(f"[+] Fetched {len(opt_rows)} prompt optimization entries from database")

        for original, optimized, template in opt_rows:
            output_obj = {
                "original_prompt": original.strip(),
                "optimized_prompt": optimized.strip(),
                "template_used": template
            }
            samples.append({
                "instruction": "Optimize the given prompt for better cognitive load analysis and emotional state extraction.",
                "input": original.strip(),
                "output": json.dumps(output_obj, ensure_ascii=False)
            })

        cursor.close()
        conn.close()
        return samples

    except Exception as e:
        print(f"[!] Database connection failed: {e}")
        print("[*] Falling back to built-in sample dataset...")
        return []


def generate(args):
    """Generate JSONL dataset from DB data + fallback samples."""
    db_samples = []

    if not args.skip_db:
        db_samples = fetch_journals_from_db(args)

    # Always include fallback samples as a baseline
    all_samples = db_samples + FALLBACK_SAMPLES

    # Deduplicate by input text
    seen_inputs = set()
    unique_samples = []
    for sample in all_samples:
        key = sample["input"].strip()
        if key not in seen_inputs:
            seen_inputs.add(key)
            unique_samples.append(sample)

    os.makedirs(os.path.dirname(args.output) if os.path.dirname(args.output) else ".", exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as f:
        for item in unique_samples:
            f.write(json.dumps(item, ensure_ascii=False) + "\n")

    print(f"\n{'=' * 60}")
    print(f"  DATASET GENERATION COMPLETE")
    print(f"{'=' * 60}")
    print(f"[+] Total samples      : {len(unique_samples)}")
    print(f"[+] From database       : {len(db_samples)}")
    print(f"[+] Built-in fallback   : {len(FALLBACK_SAMPLES)}")
    print(f"[+] Output file         : {args.output}")
    print(f"{'=' * 60}")


if __name__ == "__main__":
    args = parse_args()
    generate(args)
