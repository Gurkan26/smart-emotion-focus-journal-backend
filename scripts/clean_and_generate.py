#!/usr/bin/env python3
"""
Clean and Generate Prompt Dataset
MasterFabric Academy - AI Dataset Refinement Pipeline

Deduplicates the existing dataset, normalizes labels, and appends synthetic examples
to achieve a balanced distribution of 60 examples per template category
(totaling 360 rows). All examples are structured for XML-based self-thinking AI fine-tuning.
"""

import os
import csv

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
CSV_PATH = os.path.abspath(os.path.join(SCRIPT_DIR, "..", "prompt_dataset.csv"))

# Synthetic Examples for Self-Thinking AI
SYNTHETIC_EXAMPLES = [
    # =========================================================================
    # MINIMAL (56 examples total)
    # =========================================================================
    (
        "get current year in bash",
        "minimal",
        "<thinking>The user needs a quick bash command to print the current year. To keep it minimal and save tokens, I should output only the command itself without explanation or code blocks.</thinking> <output><role>Systems Engineer</role><task>Print the current year in Bash.</task><instructions>Provide only the exact command.</instructions><output_format>Plain command string</output_format><constraints>No explanation, no formatting, no markdown blocks.</constraints></output>"
    ),
    (
        "check port 8080",
        "minimal",
        "<thinking>The user wants to check if port 8080 is open/in use. A simple lsof or netstat command is required. Keep it strictly minimal.</thinking> <output><role>Network Administrator</role><task>Check if port 8080 is active.</task><instructions>Provide the standard Unix shell command.</instructions><output_format>Plain command string</output_format><constraints>No explanation, no markdown formatting.</constraints></output>"
    ),
    (
        "git pull force",
        "minimal",
        "<thinking>The user wants to force pull and overwrite local changes. This requires resetting to the origin branch. Direct command only.</thinking> <output><role>Git Specialist</role><task>Force pull and discard local changes.</task><instructions>Provide the sequence of git commands.</instructions><output_format>Command block</output_format><constraints>No warnings or explanation.</constraints></output>"
    ),
    (
        "delete docker volume",
        "minimal",
        "<thinking>The user needs to delete a docker volume. Direct CLI command needed, no fluff.</thinking> <output><role>DevOps Specialist</role><task>Delete a docker volume.</task><instructions>Provide the docker volume rm command.</instructions><output_format>Plain command</output_format><constraints>No text explanation.</constraints></output>"
    ),
    (
        "extract tar.gz",
        "minimal",
        "<thinking>The user wants to unpack a tar.gz file. Give the standard tar extract command.</thinking> <output><role>Linux Administrator</role><task>Extract tar.gz archive.</task><instructions>Provide the command tar -xzvf.</instructions><output_format>Command string</output_format><constraints>No explanations.</constraints></output>"
    ),
    (
        "show disk space",
        "minimal",
        "<thinking>The user wants to check free disk space on Linux. Give df command in human-readable format.</thinking> <output><role>System Administrator</role><task>Show disk space.</task><instructions>Provide the df -h command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "kill process on 3000",
        "minimal",
        "<thinking>The user needs to kill whatever is running on port 3000. Give a clean one-liner command.</thinking> <output><role>Backend Engineer</role><task>Kill process on port 3000.</task><instructions>Provide the kill or lsof command combination.</instructions><output_format>Plain command</output_format><constraints>No verbose comments.</constraints></output>"
    ),
    (
        "find large files",
        "minimal",
        "<thinking>The user wants to locate large files in Unix. Provide a command using find with size flags.</thinking> <output><role>Systems Engineer</role><task>Find files larger than 100MB.</task><instructions>Provide the find command with human-readable output sorting.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "list git branches",
        "minimal",
        "<thinking>The user wants to see all branches. Minimal git branch command.</thinking> <output><role>Git Expert</role><task>List branches.</task><instructions>Provide git branch command showing both local and remote branches.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"
    ),
    (
        "python version command",
        "minimal",
        "<thinking>The user wants the terminal command to check python version. Keep it simple.</thinking> <output><role>Developer</role><task>Check python version.</task><instructions>Provide python3 --version command.</instructions><output_format>Command</output_format><constraints>No explanations.</constraints></output>"
    ),
    (
        "convert json to yaml cli",
        "minimal",
        "<thinking>The user wants a command-line method to convert JSON to YAML. Recommend yq or a quick python one-liner.</thinking> <output><role>DevOps Engineer</role><task>Convert JSON to YAML via CLI.</task><instructions>Provide the yq command.</instructions><output_format>Command string</output_format><constraints>No text surrounding.</constraints></output>"
    ),
    (
        "base64 decode terminal",
        "minimal",
        "<thinking>The user wants to base64 decode a string in the terminal. Provide the standard echo | base64 --decode command.</thinking> <output><role>Security Analyst</role><task>Base64 decode string.</task><instructions>Provide terminal command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "curl post json",
        "minimal",
        "<thinking>The user wants to send a POST request with JSON payload using curl. Give a clean formatted curl command.</thinking> <output><role>API Developer</role><task>Send JSON POST request.</task><instructions>Provide the curl command with headers and payload placeholders.</instructions><output_format>Curl command block</output_format><constraints>No introduction.</constraints></output>"
    ),
    (
        "prettify json command",
        "minimal",
        "<thinking>The user wants to format JSON via CLI. Provide python -m json.tool or jq command.</thinking> <output><role>Developer</role><task>Format JSON.</task><instructions>Provide jq command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "restart nginx",
        "minimal",
        "<thinking>The user needs to restart Nginx. Systemctl command is appropriate.</thinking> <output><role>DevOps</role><task>Restart Nginx.</task><instructions>Provide systemctl restart nginx command.</instructions><output_format>Command</output_format><constraints>No metadata.</constraints></output>"
    ),
    (
        "check memory usage free",
        "minimal",
        "<thinking>The user wants to view RAM usage. Give the free command with human-readable flag.</thinking> <output><role>SysAdmin</role><task>Check RAM usage.</task><instructions>Provide free -h command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "get public ip cmd",
        "minimal",
        "<thinking>The user wants to find their public IP via terminal. Give curl ifconfig.me or similar.</thinking> <output><role>Systems Engineer</role><task>Get public IP.</task><instructions>Provide curl command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "find string in files recursive",
        "minimal",
        "<thinking>The user wants to search for text inside files. Provide grep command.</thinking> <output><role>Developer</role><task>Find text in files.</task><instructions>Provide grep -r command.</instructions><output_format>Command</output_format><constraints>No comments.</constraints></output>"
    ),
    (
        "create empty 1gb file",
        "minimal",
        "<thinking>The user wants to allocate a 1GB file quickly. Recommend fallocate or dd.</thinking> <output><role>SysAdmin</role><task>Create empty 1GB file.</task><instructions>Provide fallocate command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"
    ),
    (
        "chmod read only",
        "minimal",
        "<thinking>The user wants to make a file read-only. Give chmod command.</thinking> <output><role>Security Specialist</role><task>Make file read-only.</task><instructions>Provide chmod 400 or chmod 444 command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "generate ssh key",
        "minimal",
        "<thinking>The user wants to create an SSH key pair. Give ssh-keygen command.</thinking> <output><role>SecOps</role><task>Generate SSH key.</task><instructions>Provide ssh-keygen -t ed25519 command.</instructions><output_format>Command</output_format><constraints>No commentary.</constraints></output>"
    ),
    (
        "dns lookup cli",
        "minimal",
        "<thinking>The user wants to query DNS. Give dig command.</thinking> <output><role>Network Engineer</role><task>DNS lookup.</task><instructions>Provide dig command for A records.</instructions><output_format>Command</output_format><constraints>No extra text.</constraints></output>"
    ),
    (
        "check line count of file",
        "minimal",
        "<thinking>The user wants to count lines. Provide wc -l command.</thinking> <output><role>Developer</role><task>Count lines in file.</task><instructions>Provide wc -l command.</instructions><output_format>Command</output_format><constraints>No explanations.</constraints></output>"
    ),
    (
        "unzip to directory",
        "minimal",
        "<thinking>The user wants to unzip a zip archive to a specific path. Give unzip -d command.</thinking> <output><role>Linux User</role><task>Unzip to folder.</task><instructions>Provide unzip command with destination flag.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "copy directory terminal",
        "minimal",
        "<thinking>The user wants to recursively copy a folder. Provide cp -r command.</thinking> <output><role>SysAdmin</role><task>Copy directory.</task><instructions>Provide cp -r command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"
    ),
    (
        "list env variables",
        "minimal",
        "<thinking>The user wants to list all environment variables. Provide printenv or env command.</thinking> <output><role>Developer</role><task>List environment variables.</task><instructions>Provide printenv command.</instructions><output_format>Command</output_format><constraints>No extra text.</constraints></output>"
    ),
    # Additional minimal
    ("get folder size linux", "minimal", "<thinking>The user wants to check folder size on Linux. Give du command human readable.</thinking> <output><role>Systems Engineer</role><task>Get folder size.</task><instructions>Provide du -sh command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("check active connections netstat", "minimal", "<thinking>User needs active network connections. Provide netstat command.</thinking> <output><role>Network Engineer</role><task>Check connections.</task><instructions>Provide netstat -an command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("git restore file", "minimal", "<thinking>User wants to revert changes in a file in git. Give git restore command.</thinking> <output><role>Git Expert</role><task>Revert file changes.</task><instructions>Provide git restore command.</instructions><output_format>Command</output_format><constraints>No commentary.</constraints></output>"),
    ("delete docker container by name", "minimal", "<thinking>User needs to remove a docker container. Give docker rm.</thinking> <output><role>DevOps</role><task>Remove container.</task><instructions>Provide docker rm command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("list listening ports linux", "minimal", "<thinking>User wants to see which ports are listening. Provide ss command.</thinking> <output><role>Network Analyst</role><task>List listening ports.</task><instructions>Provide ss -lntp command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("show memory free megabytes", "minimal", "<thinking>User wants memory in MB. Provide free -m.</thinking> <output><role>SysAdmin</role><task>Show RAM in MB.</task><instructions>Provide free -m command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("kill all node processes", "minimal", "<thinking>User wants to kill all Node.js processes. Provide killall.</thinking> <output><role>Developer</role><task>Kill node processes.</task><instructions>Provide killall node command.</instructions><output_format>Command</output_format><constraints>No commentary.</constraints></output>"),
    ("find directories only", "minimal", "<thinking>User wants to search directories only. Give find with type d.</thinking> <output><role>SysAdmin</role><task>Find directories.</task><instructions>Provide find command with -type d.</instructions><output_format>Command</output_format><constraints>No extra text.</constraints></output>"),
    ("show git status short", "minimal", "<thinking>User wants compact git status. Provide git status -s.</thinking> <output><role>Git Lead</role><task>Short git status.</task><instructions>Provide git status -s command.</instructions><output_format>Command</output_format><constraints>No warnings.</constraints></output>"),
    ("python virtualenv create", "minimal", "<thinking>User wants to create a virtual environment in python. Provide venv module invocation.</thinking> <output><role>Python Dev</role><task>Create virtualenv.</task><instructions>Provide python3 -m venv command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("convert yaml to json python", "minimal", "<thinking>User wants quick CLI yaml to json conversion. Provide python one-liner.</thinking> <output><role>Developer</role><task>Convert YAML to JSON.</task><instructions>Provide python one-liner using yaml and json modules.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("url encode string bash", "minimal", "<thinking>User wants to url encode text in bash. Provide command template.</thinking> <output><role>Developer</role><task>URL encode.</task><instructions>Provide raw python or curl string utility.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("curl check status code", "minimal", "<thinking>User wants to query only the HTTP status code. Give curl command.</thinking> <output><role>DevOps</role><task>Get HTTP status code.</task><instructions>Provide curl command with custom output format.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("prettify xml terminal", "minimal", "<thinking>User wants to format XML. Give xmllint.</thinking> <output><role>SysAdmin</role><task>Format XML.</task><instructions>Provide xmllint --format command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("reload systemd daemon", "minimal", "<thinking>User wants to reload systemd configurations. Give daemon-reload.</thinking> <output><role>SysAdmin</role><task>Reload systemd.</task><instructions>Provide systemctl daemon-reload command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("check cpu temperature cmd", "minimal", "<thinking>User wants CPU temp. Give sensors.</thinking> <output><role>SysAdmin</role><task>Get CPU temp.</task><instructions>Provide sensors command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("get local ip unix", "minimal", "<thinking>User wants local IP. Give hostname.</thinking> <output><role>SysAdmin</role><task>Get local IP.</task><instructions>Provide hostname -I command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("find text case insensitive", "minimal", "<thinking>User wants case insensitive search in files. Give grep.</thinking> <output><role>Developer</role><task>Search text case insensitive.</task><instructions>Provide grep -ri command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("create file with timestamp", "minimal", "<thinking>User wants file named with date. Give touch/date command.</thinking> <output><role>SysAdmin</role><task>Create dated file.</task><instructions>Provide touch command with subshell execution of date.</instructions><output_format>Command</output_format><constraints>No comments.</constraints></output>"),
    ("chmod execute permission", "minimal", "<thinking>User wants to make a script executable. Give chmod +x.</thinking> <output><role>Developer</role><task>Make executable.</task><instructions>Provide chmod +x command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("view ssh key fingerprint", "minimal", "<thinking>User wants key fingerprint. Give ssh-keygen.</thinking> <output><role>Security Analyst</role><task>Get SSH fingerprint.</task><instructions>Provide ssh-keygen -l -f command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("lookup mx records", "minimal", "<thinking>User wants MX record lookup. Give dig.</thinking> <output><role>Network Engineer</role><task>MX lookup.</task><instructions>Provide dig MX command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("count occurrences of string in file", "minimal", "<thinking>User wants count of target string. Give grep and wc.</thinking> <output><role>Developer</role><task>Count occurrences.</task><instructions>Provide grep -o | wc -l command sequence.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("extract zip file", "minimal", "<thinking>User wants to extract zip. Give unzip.</thinking> <output><role>User</role><task>Extract zip.</task><instructions>Provide unzip command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("rsync copy progressive", "minimal", "<thinking>User wants to transfer folders showing progress. Give rsync.</thinking> <output><role>SysAdmin</role><task>Rsync transfer.</task><instructions>Provide rsync -avP command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("list aliases linux", "minimal", "<thinking>User wants active shell aliases. Give alias.</thinking> <output><role>User</role><task>List aliases.</task><instructions>Provide alias command.</instructions><output_format>Command</output_format><constraints>No commentary.</constraints></output>"),
    ("check linux kernel version", "minimal", "<thinking>User wants kernel release. Give uname.</thinking> <output><role>SysAdmin</role><task>Check kernel version.</task><instructions>Provide uname -r command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),
    ("clear dns cache mac", "minimal", "<thinking>User wants to flush dns on Mac. Give dscacheutil.</thinking> <output><role>SysAdmin</role><task>Flush DNS Mac.</task><instructions>Provide dscacheutil -flushcache command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("git show last commit hash", "minimal", "<thinking>User wants last commit hash. Give git rev-parse.</thinking> <output><role>Git Lead</role><task>Show commit hash.</task><instructions>Provide git rev-parse HEAD command.</instructions><output_format>Command</output_format><constraints>No explanation.</constraints></output>"),
    ("run bash script background", "minimal", "<thinking>User wants to launch script in background. Give nohup.</thinking> <output><role>Developer</role><task>Run background script.</task><instructions>Provide nohup ./script.sh & command.</instructions><output_format>Command</output_format><constraints>No text.</constraints></output>"),

    # =========================================================================
    # CREATIVE (56 examples total)
    # =========================================================================
    (
        "write sci fi story",
        "creative",
        "<thinking>The user wants a science fiction story. I need to guide the model to establish a rich narrative, introduce a speculative concept, focus on sensory details, and ensure character-driven conflict.</thinking> <output><role>Sci-Fi Novelist & Futurist</role><context>A distant solar system where organic consciousness is synthesized.</context><task>Write an immersive sci-fi short story.</task><instructions>Develop a character facing an existential dilemma related to synthetic memories. Use vivid descriptions and scientific metaphors. Keep the pace deliberate and end with a philosophical resolution.</instructions><output_format>Narrative prose with title and structured scene transitions.</output_format><constraints>Avoid generic tropes like space battles. Focus on character depth.</constraints></output>"
    ),
    (
        "slogan for organic tea",
        "creative",
        "<thinking>The user needs a catchy, natural brand slogan for organic tea. It should evoke calm, health, and purity. Guide the model to provide diverse creative options.</thinking> <output><role>Brand Strategist</role><context>Eco-conscious consumers looking for daily wellness rituals.</context><task>Create slogans for organic herbal tea.</task><instructions>Brainstorm 5 unique slogans. Categorize them by tone (e.g., poetic, direct, minimalist). Explain the emotion behind each concept.</instructions><output_format>Categorized list with brief conceptual explanations.</output_format><constraints>Avoid cliché terms like 'natural taste' without context.</constraints></output>"
    ),
    (
        "name for fintech startup",
        "creative",
        "<thinking>Fintech startup naming requires a balance of trust, innovation, and modern appeal. The prompt should ask for varied conceptual angles.</thinking> <output><role>Naming Consultant & Brand Architect</role><context>A mobile-first global payment network using decentralized ledgers.</context><task>Generate names for a new fintech brand.</task><instructions>Provide 8 name suggestions organized by theme (e.g., trust-based, abstract, compound names). For each name, provide its phonetic feel and target brand personality.</instructions><output_format>Structured naming brief with tables or bullet lists.</output_format><constraints>Ensure names are short, easy to pronounce globally, and feel premium.</constraints></output>"
    ),
    (
        "plot hook for fantasy",
        "creative",
        "<thinking>The user wants a fantasy plot hook. I need to prompt for high-concept conflict, avoiding standard tavern starts, focusing on magic systems and stakes.</thinking> <output><role>Fantasy Worldbuilder & Story Consultant</role><context>A world where magic is a decaying, non-renewable natural resource.</context><task>Write an intriguing fantasy story starter.</task><instructions>Establish a major conflict in 3 paragraphs. Introduce a unique magical cost, a protagonist with conflicting loyalties, and a ticking clock element.</instructions><output_format>Atmospheric narrative introduction.</output_format><constraints>Do not write a full story; focus entirely on the hook and stakes.</constraints></output>"
    ),
    (
        "describe a cyberpunk city",
        "creative",
        "<thinking>Cyberpunk requires neon-drenched, high-tech/low-life contrast, rich sensory descriptions, and structural metaphors (verticality). Let's direct the model to construct a vivid scene.</thinking> <output><role>World Designer & Concept Artist</role><task>Write a detailed sensory description of a cyberpunk metropolis.</task><instructions>Describe the city through the senses: sight (holographic rain, neon decay), sound (drone hum, street vendors), and smell (smog, synthetic street food). Highlight the contrast between the lower slums and upper high-tech spires.</instructions><output_format>Immersive descriptive essay split into sensory layers.</output_format><constraints>Avoid flat adjectives; use active verbs and tech-industrial metaphors.</constraints></output>"
    ),
    (
        "character description of a rogue",
        "creative",
        "<thinking>The user wants a rogue description. Avoid standard 'cloaked in shadows' clichés; require psychological depth, distinct physical tells, and gear with history.</thinking> <output><role>RPG Narrative Designer</role><task>Draft a character profile for a master thief.</task><instructions>Include physical appearance, a subtle tell that reveals when they are lying, their weapon's unique origin, and their driving fear.</instructions><output_format>Character dossier format with subheadings.</output_format><constraints>No generic fantasy rogue tropes. Focus on subversion of expectations.</constraints></output>"
    ),
    (
        "blog post title ideas for productivity",
        "creative",
        "<thinking>Productivity titles need to promise value while remaining click-worthy but not clickbaity. Guide the model to generate multiple psychological angles.</thinking> <output><role>Content Strategist & Copywriter</role><task>Generate high-converting blog post titles about workspace productivity.</task><instructions>Create 10 titles divided by style: Curiosity-Gap, How-To, Data-Driven, and Counter-Intuitive. Explain why each works.</instructions><output_format>Markdown list with analysis notes.</output_format><constraints>No hype-laden titles. Keep them professional and realistic.</constraints></output>"
    ),
    (
        "write a poem about debugging",
        "creative",
        "<thinking>A poem about debugging should balance technical accuracy (logs, stacks, memory leaks) with poetic form and emotional resonance (frustration, triumph).</thinking> <output><role>Tech Poet Laureate</role><task>Compose a poem about debugging an elusive bug.</task><instructions>Write a 4-stanza poem using metaphors of hunting, labyrinths, or archaeology. Integrate actual coding concepts subtly. Maintain a consistent meter.</instructions><output_format>Poetic stanzas.</output_format><constraints>Do not make it overly silly; focus on the quiet focus and sudden realization of coding.</constraints></output>"
    ),
    (
        "dialogue between cat and dog",
        "creative",
        "<thinking>The user wants a dialogue. To make it creative, let's frame them as philosophers debating the nature of loyalty vs independence, avoiding simple cartoons.</thinking> <output><role>Playwright & Scriptwriter</role><task>Write a philosophical dialogue between a household cat and dog.</task><instructions>Write a dialogue where the dog argues for the beauty of obedience and duty, while the cat defends autonomy and skepticism. Keep their behaviors animalistic but their thoughts profound.</instructions><output_format>Script format (CAT: ..., DOG: ...).</output_format><constraints>No slapstick comedy. Use subtext and quiet tension.</constraints></output>"
    ),
    (
        "brand identity for coffee shop",
        "creative",
        "<thinking>Coffee shop brand identity needs to capture visual aesthetic, values, and community tone. Require concrete design elements.</thinking> <output><role>Creative Director</role><context>A slow-drip coffee shop catering to writers and local artists.</context><task>Design a comprehensive brand identity framework.</task><instructions>Define the brand values, color palette (with hex codes/inspirations), typography style, and the signature 'vibe' of the physical space.</instructions><output_format>Brand book chapter outline.</output_format><constraints>Do not design a logo; define the creative parameters that guide a designer.</constraints></output>"
    ),
    (
        "concept for a futuristic museum",
        "creative",
        "<thinking>A futuristic museum concept should explore high-tech interactive exhibits and speculative curation (e.g., historical artifacts of the early digital era).</thinking> <output><role>Experience Designer & Futurist</role><task>Develop a concept for the 'Museum of Extinct Technologies'.</task><instructions>Describe 3 key exhibits (e.g., a physical smartphone, a mechanical hard drive) and how futuristic visitors interact with them using neural links or holographic projection.</instructions><output_format>Curator's exhibition guide.</output_format><constraints>Ensure the tone is educational yet slightly melancholic about the past.</constraints></output>"
    ),
    (
        "tagline for a meditation app",
        "creative",
        "<thinking>Meditation app taglines must communicate calm, focus, and clarity in very few words. Guide the model to explore linguistic patterns.</thinking> <output><role>Linguistic Copywriter</role><task>Generate taglines for a mindfulness application.</task><instructions>Write 6 taglines exploring different styles: alliteration, paradox, ultra-short, and verb-led. Explain the cognitive impact of each phrase.</instructions><output_format>Analytical brand copy proposal.</output_format><constraints>Avoid cliché phrases like 'breathe in, breathe out'.</constraints></output>"
    ),
    (
        "describe a magical artifact",
        "creative",
        "<thinking>A magical artifact needs physical weight, history, and a double-edged cost to its power to feel interesting and original.</thinking> <output><role>Fantasy Lore Designer</role><task>Describe a relic known as the 'Hourglass of Unspent Grief'.</task><instructions>Describe its material, how it is activated, the magic it commands (reversing time by reliving painful memories), and the physiological cost of using it.</instructions><output_format>Item description for a dark fantasy setting.</output_format><constraints>Avoid generic 'overpowered' magic; the artifact must have a distinct limitation and cost.</constraints></output>"
    ),
    (
        "speech outline for new year",
        "creative",
        "<thinking>A speech outline needs structured rhetorical devices (anaphora, contrast), emotional pacing, and a clear unifying metaphor.</thinking> <output><role>Executive Speechwriter</role><context>An address to a tech company's employees looking back at a hard year and forward to change.</context><task>Write a speech outline and opening hook.</task><instructions>Provide an outline with timing, key themes, emotional targets, and the full text of the first 2 minutes of the speech using a strong central metaphor.</instructions><output_format>Rhetorical speech brief.</output_format><constraints>Do not write a generic motivational speech. Root it in realistic challenges.</constraints></output>"
    ),
    (
        "brainstorm name for workspace tool",
        "creative",
        "<thinking>Names for digital collaboration tools should evoke clean workspace organization, seamlessness, and focus.</thinking> <output><role>Creative Branding Specialist</role><context>A tool that merges markdown notes with calendar events.</context><task>Brainstorm names for a calendar-note productivity app.</task><instructions>Suggest 6 names based on nature, geometry, or classical concepts. Include domain availability strategy and logo concept sketch description for each.</instructions><output_format>Structured naming presentation.</output_format><constraints>Avoid appending '-ly' or '-ify' to words; choose organic or compound terms.</constraints></output>"
    ),
    (
        "sci-fi weapon description",
        "creative",
        "<thinking>A sci-fi weapon description should be grounded in speculative physics and operational detail to make it believable, rather than just magic lasers.</thinking> <output><role>Weapons System Concept Designer</role><task>Describe the mechanics of a localized gravity-well projector.</task><instructions>Explain how the weapon draws power, its physical interface, the visual and physical disruption caused when fired, and its tactical drawbacks.</instructions><output_format>Technical manual entry with a narrative introduction.</output_format><constraints>Do not write pure action prose; focus on design, physics, and ergonomics.</constraints></output>"
    ),
    (
        "idea for a puzzle game mechanic",
        "creative",
        "<thinking>A game mechanic should be easy to understand but offer high emergent complexity. Guide the model to define the gameplay loop.</thinking> <output><role>Lead Game Designer</role><task>Design a spatial puzzle mechanic based on shifting gravity.</task><instructions>Define the core rules, control inputs, how it generates complexity across levels, and a sample puzzle scenario.</instructions><output_format>Game design document (GDD) snippet.</output_format><constraints>Ensure the mechanic is original and explainable in 3 simple rules.</constraints></output>"
    ),
    (
        "social media caption for travel",
        "creative",
        "<thinking>Travel captions on social media often sound generic. Guide the model to build an micro-narrative that tells a small story rather than just using emojis.</thinking> <output><role>Social Media Storyteller</role><context>A photo of a hidden mountain lake at sunrise after a long hike.</context><task>Write an engaging micro-narrative caption.</task><instructions>Provide 3 options: an evocative story-driven caption (80 words), a minimalist reflection (15 words), and a curiosity hook (30 words).</instructions><output_format>Captions with targeted hashtags and formatting.</output_format><constraints>No overly optimistic travel blogger platitudes. Keep it grounded.</constraints></output>"
    ),
    (
        "describe a hidden forest temple",
        "creative",
        "<thinking>A hidden temple should feel ancient, overgrown, and sacred. Focus on light, architecture merging with nature, and silent atmosphere.</thinking> <output><role>Environmental Concept Writer</role><task>Describe a ruined sanctuary inside an ancient redwood forest.</task><instructions>Focus on the relationship between stone and roots. Describe how light breaks through the canopy, the wildlife that has claimed the space, and the emotional effect of the silence.</instructions><output_format>Atmospheric descriptive text.</output_format><constraints>Avoid generic fantasy combat hints. Focus on tranquility and decay.</constraints></output>"
    ),
    (
        "write an elevator pitch for VR gym",
        "creative",
        "<thinking>An elevator pitch needs a hook, a clear problem statement (boredom in cardio), a solution (gamified VR fitness), and a target demographic.</thinking> <output><role>Pitch Coach & Startup Advisor</role><task>Draft a 60-second elevator pitch for a VR fitness franchise.</task><instructions>Structure the pitch into: The Hook, The Pain Point, The Innovation, The Business Case, and The Call to Action. Use active, persuasive language.</instructions><output_format>Spoken-word script format with physical cue suggestions.</output_format><constraints>Keep the word count under 150 words to fit a 60-second delivery.</constraints></output>"
    ),
    (
        "concept for a time-travel movie",
        "creative",
        "<thinking>Time travel concepts are often plagued by paradoxes. A creative twist should focus on the psychological impact of travel, not just technology.</thinking> <output><role>Screenwriter & Director</role><task>Pitch a high-concept time-travel film.</task><instructions>Outline the central premise (e.g., travel is one-way and you only go back 5 minutes), the protagonist's core conflict, and a dramatic turning point.</instructions><output_format>Film pitch deck executive summary.</output_format><constraints>Avoid typical save-the-world time travel plots. Focus on localized, intimate stakes.</constraints></output>"
    ),
    (
        "intro for a podcast about history",
        "creative",
        "<thinking>A podcast intro needs a compelling hook, setting of the show's theme, and sonic pacing notes.</thinking> <output><role>Podcast Producer & Copywriter</role><context>A narrative history podcast exploring the forgotten failures of famous inventors.</context><task>Write the introductory script for episode 1.</task><instructions>Create a script that starts with a dramatic scene-setting, introduces the show's premise, and has cues for music swells and sound effects.</instructions><output_format>Audio script with sound direction cues.</output_format><constraints>Keep the tone mysterious and intellectual, avoiding standard radio announcer styles.</constraints></output>"
    ),
    (
        "tagline for a sustainable fashion brand",
        "creative",
        "<thinking>Sustainable fashion taglines need to avoid greenwashing and instead promise quality, durability, and modern style.</thinking> <output><role>Sustainability Copywriter</role><task>Create brand taglines for organic cotton apparel.</task><instructions>Provide 5 options highlighting different value propositions (e.g., durability, minimal footprint, timeless design). Explain the positioning of each.</instructions><output_format>Copywriting options list.</output_format><constraints>Do not use clichés like 'save the planet' or 'eco-friendly'.</constraints></output>"
    ),
    (
        "describe a solar punk community",
        "creative",
        "<thinking>Solar punk focuses on a sustainable, technologically integrated future. Describe the harmony between high tech (solar glass, community grids) and nature.</thinking> <output><role>Utopian Fiction Author</role><task>Describe a day in the life of a solar punk neighborhood.</task><instructions>Focus on the community infrastructure, food systems, local governance, and visual integration of solar technology with architectural flora.</instructions><output_format>Narrative description.</output_format><constraints>Avoid dystopian warnings; focus entirely on active, healthy adaptation.</constraints></output>"
    ),
    (
        "write a metaphor for memory",
        "creative",
        "<thinking>A good metaphor should be original and visually rich. Avoid basic 'filing cabinet' metaphors.</thinking> <output><role>Creative Writing Instructor</role><task>Draft an extended metaphor comparing memory to a tides on a dynamic shore.</task><instructions>Explain how memories wash up, get eroded by water, leave residues, and change the shape of the coastline over time. Use lyrical prose.</instructions><output_format>Short essay or prose poem.</output_format><constraints>Avoid standard computer storage analogies.</constraints></output>"
    ),
    (
        "dialogue between two astronauts lost",
        "creative",
        "<thinking>This scenario requires intense subtext, high stakes, and realistic operational terminology. Avoid over-dramatized panic; use professional focus to mask underlying fear.</thinking> <output><role>Dramatist & Playwright</role><context>Two astronauts adrift in a failing capsule with 20 minutes of oxygen left.</context><task>Write a dramatic scene dialogue.</task><instructions>Show them trying to calculate orbital vectors while talking about what they will miss on Earth. Keep the dialogue realistic, professional, and emotionally restrained.</instructions><output_format>Screenplay format.</output_format><constraints>Do not write hysterics. The tension must come from the quiet execution of procedures despite the inevitable.</constraints></output>"
    ),
    # Additional creative
    ("describe a fantasy tavern", "creative", "<thinking>User wants descriptive environment. Focus on tavern smells, lighting, noise, and patrons.</thinking> <output><role>Fantasy Worldbuilder</role><task>Describe a bustling fantasy tavern.</task><instructions>Detail the crackling hearth, the scent of roasted meats and stale ale, the murmur of travelers, and the dim firelight casting shadows.</instructions><output_format>Atmospheric prose paragraph.</output_format><constraints>No combat or plot events. Focus entirely on atmosphere.</constraints></output>"),
    ("slogan for virtual reality headset", "creative", "<thinking>VR headset slogan. Needs to focus on immersive experience, alternate realities.</thinking> <output><role>Tech Marketer</role><task>Create slogans for VR headset.</task><instructions>Brainstorm 3 tagline concepts focusing on boundary-less vision.</instructions><output_format>List of slogans.</output_format><constraints>Avoid generic terms like 'best visual experience'.</constraints></output>"),
    ("name for an eco-friendly cleaning brand", "creative", "<thinking>Eco cleaning brand naming. Evoke green, cleanliness, safety, science.</thinking> <output><role>Naming Consultant</role><task>Suggest names for green cleaner brand.</task><instructions>Provide 5 options combining natural roots with active clean sounds.</instructions><output_format>Name list with concepts.</output_format><constraints>Ensure easy pronunciation.</constraints></output>"),
    ("plot hook for detective noir", "creative", "<thinking>Detective noir needs rain-slicked streets, a mysterious client, a moral grey area.</thinking> <output><role>Noir Writer</role><task>Draft detective noir plot hook.</task><instructions>Write a scene where a shadow client arrives at midnight with an item that shouldn't exist.</instructions><output_format>Intro scene dialogue/prose.</output_format><constraints>Maintain typical cynical tone.</constraints></output>"),
    ("describe a solar eclipse over an alien ocean", "creative", "<thinking>Alien environment description. Focus on color palettes (e.g. purple seas, binary suns), gravitational pulls.</thinking> <output><role>Sci-Fi Concept Writer</role><task>Describe alien solar eclipse.</task><instructions>Detail the shifting light, behavior of bioluminescent marine life, and changes in tide.</instructions><output_format>Descriptive passage.</output_format><constraints>Focus on sensory details of alien nature.</constraints></output>"),
    ("character description of a steampunk inventor", "creative", "<thinking>Steampunk inventor needs brass goggles, soot-stained clothes, ticks related to gears.</thinking> <output><role>Steampunk Novelist</role><task>Describe an inventor character.</task><instructions>Detail physical description, tools worn, and a nervous habit related to pocket watches.</instructions><output_format>Character sheet snippet.</output_format><constraints>Avoid modern tech terms; restrict to steam, hydraulics, and clockwork.</constraints></output>"),
    ("blog post title ideas for healthy cooking", "creative", "<thinking>Cooking blog titles. Needs to feel accessible, quick, and nutritious.</thinking> <output><role>Food Content Specialist</role><task>Create healthy recipe blog titles.</task><instructions>Brainstorm 6 title options focusing on time limits and flavor preservation.</instructions><output_format>Title ideas list.</output_format><constraints>No spammy clickbait.</constraints></output>"),
    ("write a sonnet about black holes", "creative", "<thinking>Sonnet requires specific structure (ABAB CDCD EFEF GG in iambic pentameter). Theme: gravity, singularity.</thinking> <output><role>Poet</role><task>Compose sonnet on black holes.</task><instructions>Write a Shakespearean sonnet describing the event horizon and light capture.</instructions><output_format>Poem lines.</output_format><constraints>Strictly adhere to 14 lines and standard rhyme scheme.</constraints></output>"),
    ("dialogue between a clock and an hourglass", "creative", "<thinking>Dialogue between mechanical time (ticks) and fluid time (flowing sand).</thinking> <output><role>Creative Playwright</role><task>Write a dialogue about time's passing.</task><instructions>Show the tension between mechanical precision and natural accumulation/loss.</instructions><output_format>Script format.</output_format><constraints>Keep it brief and poetic.</constraints></output>"),
    ("brand identity for a zero-waste grocery store", "creative", "<thinking>Zero waste grocery. Core values: zero packaging, circular supply, community weight scaling.</thinking> <output><role>Brand Identity Designer</role><task>Design identity document outline.</task><instructions>Define values, typography style, visual packaging guidelines for reusable containers.</instructions><output_format>Brand parameters brief.</output_format><constraints>No specific logo design; focus on conceptual brand rules.</constraints></output>"),
    ("concept for a high-tech library", "creative", "<thinking>High tech library. Interactive memory retrieval, holographic historical debates.</thinking> <output><role>Experience Designer</role><task>Describe futuristic library exhibit.</task><instructions>Detail how patrons interact with stored memories using spatial audio and dynamic holographic actors.</instructions><output_format>Exhibit brief.</output_format><constraints>Ensure a respectful, learning-oriented atmosphere.</constraints></output>"),
    ("tagline for a language learning platform", "creative", "<thinking>Language tagline. Needs to focus on cultural immersion and conversational confidence.</thinking> <output><role>Linguistic Copywriter</role><task>Write language learning taglines.</task><instructions>Provide 4 distinct copy lines targeting business and travel markets.</instructions><output_format>Copy list.</output_format><constraints>Avoid boring educational slogans.</constraints></output>"),
    ("describe a legendary sword forged in starlight", "creative", "<thinking>Legendary weapon description. Focus on visual texture (milky obsidian blade), cosmic cold, weight.</thinking> <output><role>Fantasy Worldbuilder</role><task>Describe the sword 'Astra'.</task><instructions>Detail its forged history, properties under moonlit skies, and physical details of the hilt.</instructions><output_format>Lore entry.</output_format><constraints>Maintain mythological tone.</constraints></output>"),
    ("opening paragraph for a horror story", "creative", "<thinking>Horror opening. Needs to build instant tension through sound or spatial displacement, not jump scares.</thinking> <output><role>Horror Author</role><task>Write story opening.</task><instructions>Focus on a sound that starts inside a locked wall at 3 AM. Establish narrator panic.</instructions><output_format>Short narrative prose.</output_format><constraints>Avoid visible monsters; build suspense purely on auditory cues.</constraints></output>"),
    ("brainstorm name for a group travel planner app", "creative", "<thinking>Group travel names. Needs to sound collaborative, dynamic, adventure-bound.</thinking> <output><role>Naming Consultant</role><task>Create names for travel planner.</task><instructions>Suggest 5 names reflecting coordination, route mapping, and shared horizons.</instructions><output_format>Name list with descriptions.</output_format><constraints>Ensure domains are plausible.</constraints></output>"),
    ("describe a bio-luminescent cave system", "creative", "<thinking>Underground space description. Focus on phosphorescent moss, echoing water, cold moisture.</thinking> <output><role>Environmental Designer</role><task>Describe a crystal cave.</task><instructions>Detail the glowing blue hues, reflection on dark waterpools, and smell of wet limestone.</instructions><output_format>Descriptive passage.</output_format><constraints>Focus on botanical and physical environment, no combat.</constraints></output>"),
    ("idea for a cooperative card game mechanic", "creative", "<thinking>Card game mechanic. Shared hand limit, hidden clues, non-verbal communication rules.</thinking> <output><role>Game Designer</role><task>Draft game mechanic rules.</task><instructions>Detail how players must coordinate without speaking by matching card symbols to shared paths.</instructions><output_format>Rules snippet.</output_format><constraints>Ensure rules are logically sound.</constraints></output>"),
    ("social media caption for a cozy bookstore afternoon", "creative", "<thinking>Cozy bookstore caption. Evoke smell of paper, warm tea, escaping rain.</thinking> <output><role>Social Copywriter</role><task>Write bookstore caption options.</task><instructions>Provide 3 formats: narrative reflection, micro-poetry, and conversational hook.</instructions><output_format>Social post options.</output_format><constraints>No standard influencer emojis; keep it grounded.</constraints></output>"),
    ("describe a floating island city", "creative", "<thinking>High fantasy city description. Focus on dynamic suspension, cloud docks, structural gravity fields.</thinking> <output><role>Concept Architect</role><task>Describe the sky city 'Vael'.</task><instructions>Detail the harbor where skyships dock and the aqueducts that spill water to the earth below.</instructions><output_format>Atmospheric overview.</output_format><constraints>Keep physical layout coherent.</constraints></output>"),
    ("write an elevator pitch for a personal helper drone", "creative", "<thinking>Helper drone pitch. Solve time management and domestic multi-tasking.</thinking> <output><role>Tech Pitch Coach</role><task>Draft product pitch.</task><instructions>Structure as Hook, Problem, Solution, and CTA for a consumer electronics launch.</instructions><output_format>60-second speech script.</output_format><constraints>Limit word count to 130.</constraints></output>"),
    ("concept for an alternative history TV series", "creative", "<thinking>Alt history premise. What if printing press was banned in 15th-century Europe?</thinking> <output><role>Showrunner</role><task>Draft show Bible premise.</task><instructions>Outline the underground illegal printing guilds and the central inquisitor antagonist.</instructions><output_format>Series logline and outline.</output_format><constraints>Focus on social and technology implications.</constraints></output>"),
    ("intro for a podcast about architecture history", "creative", "<thinking>Podcast intro. Frame structures as physical memories of long-dead societies.</thinking> <output><role>Podcast Director</role><task>Write episode intro script.</task><instructions>Detail scene of a Roman bricklayer, transition to modern high-rises, cue music shift.</instructions><output_format>Audio script format.</output_format><constraints>Tone must be intellectual and narrative.</constraints></output>"),
    ("tagline for a modular furniture line", "creative", "<thinking>Modular furniture copy. Focus on space optimization, adaptability, fluid layouts.</thinking> <output><role>Design Copywriter</role><task>Write modular desk taglines.</task><instructions>Provide 4 taglines addressing compact home offices.</instructions><output_format>Copy proposal.</output_format><constraints>Avoid generic terms like 'multifunctional'.</constraints></output>"),
    ("describe a community living inside a massive hollow redwood tree", "creative", "<thinking>Environmental living concept. Tree trunk architecture, rope bridges, bark windows.</thinking> <output><role>World Designer</role><task>Describe tree community.</task><instructions>Detail the vertical spiraling stairs, how sap is utilized for energy, and wood carving aesthetics.</instructions><output_format>Description.</output_format><constraints>Focus on architectural integration with tree health.</constraints></output>"),
    ("write a metaphor for time", "creative", "<thinking>Metaphor for time. Compare time to a silent sculptor carving mountains.</thinking> <output><role>Creative Writing Coach</role><task>Draft time metaphor essay.</task><instructions>Explore how time shapes identity and memory through erosion and quiet accumulation.</instructions><output_format>Short essay.</output_format><constraints>No mechanical watch comparisons.</constraints></output>"),
    ("dialogue between a human and an AI therapist", "creative", "<thinking>Dialogue. Human sharing chronic stress, AI validating emotional states using analytical empathy.</thinking> <output><role>Screenplay Writer</role><task>Write conversation scene.</task><instructions>Show the contrast between raw human emotion and structured, calm AI prompts.</instructions><output_format>Script format.</output_format><constraints>Avoid standard clinical clinical terminology; make it philosophical.</constraints></output>"),
    ("describe a gothic castle during a thunderstorm", "creative", "<thinking>Gothic environment. Lightning illuminating stone gargoyles, howling wind, dark corridors.</thinking> <output><role>Atmospheric Novelist</role><task>Describe castle interior.</task><instructions>Focus on the play of shadows during lightning flashes and the smell of ozone and wet dust.</instructions><output_format>Narrative description.</output_format><constraints>No action; pure landscape/architectural focus.</constraints></output>"),
    ("slogan for a mechanical watchmaker", "creative", "<thinking>Mechanical watch watchmaker slogan. Focus on engineering history, escape from digital screens.</thinking> <output><role>Brand Specialist</role><task>Create watch slogans.</task><instructions>Develop 3 slogans celebrating gear coordination and heritage.</instructions><output_format>List.</output_format><constraints>No battery/digital references.</constraints></output>"),
    ("describe the sound of a rain forest waking up", "creative", "<thinking>Auditory sensory description. Dripping water, early avian calls, rustling leaves, growing heat hum.</thinking> <output><role>Sound Designer</role><task>Write sonic landscape description.</task><instructions>Break down the sounds chronologically from pre-dawn fog to full sun exposure.</instructions><output_format>Auditory descriptive journal.</output_format><constraints>Focus exclusively on sound details.</constraints></output>"),
    ("write a short fable about a lazy beaver", "creative", "<thinking>Fable format. Lazy beaver attempts to build dam using ready-made branches, learns structural integrity requires personal labor.</thinking> <output><role>Children's Book Author</role><task>Write short moral fable.</task><instructions>Write a 4-paragraph story ending with a clear, traditional moral maxim.</instructions><output_format>Traditional fable prose.</output_format><constraints>Maintain simple, allegorical style.</constraints></output>"),

    # =========================================================================
    # ACCURATE (53 examples total)
    # =========================================================================
    (
        "explain carbon tax",
        "accurate",
        "<thinking>The user wants an accurate explanation of a carbon tax. To ensure rigor and objectivity, the model should explain the economic logic, pigouvian taxation, implementation methods, and typical policy trade-offs.</thinking> <output><role>Environmental Economist</role><task>Explain the mechanism and economic impact of a carbon tax.</task><instructions>1. Define the concept of negative externalities and how a carbon tax internalizes them. 2. Compare carbon tax directly to cap-and-trade systems. 3. Detail the economic effect on consumers and businesses. 4. Discuss revenue recycling strategies. Cite consensus views from economists.</instructions><output_format>Structured policy briefing with clear headers.</output_format><constraints>Maintain strict economic neutrality. Do not advocate for or against; describe empirical outcomes.</constraints></output>"
    ),
    (
        "how does a diesel engine work",
        "accurate",
        "<thinking>The explanation must be thermodynamically and mechanically accurate. Focus on compression ignition vs spark ignition, 4-stroke cycle phases, and fuel injection physics.</thinking> <output><role>Mechanical Engineer</role><task>Explain the mechanical operation of a 4-stroke diesel engine.</task><instructions>1. Compare compression ignition with petrol engines. 2. Describe the 4 strokes: intake, compression, power, exhaust. 3. Explain the role of fuel injectors, air density, and thermodynamic principles (Diesel cycle). 4. Detail efficiency advantages and emissions trade-offs.</instructions><output_format>Step-by-step technical explanation with mechanical details.</output_format><constraints>Maintain scientific precision. Avoid oversimplifications that compromise mechanics.</constraints></output>"
    ),
    (
        "what is the prisoner's dilemma",
        "accurate",
        "<thinking>Game theory concept. Must explain payoff matrix, Nash equilibrium, dominant strategies, and Pareto efficiency accurately.</thinking> <output><role>Game Theory Professor</role><task>Explain the Prisoner's Dilemma formally.</task><instructions>1. Define the parameters of the game (choices, players, payoffs). 2. Present a clear payoff matrix. 3. Explain why mutual defection is the Nash equilibrium. 4. Discuss the tension between individual rationality and collective Pareto efficiency.</instructions><output_format>Academic primer with payoff matrix representation.</output_format><constraints>Ensure mathematical terms (equilibrium, dominant strategy) are used correctly.</constraints></output>"
    ),
    (
        "explain inflation causes",
        "accurate",
        "<thinking>The causes of inflation are multi-faceted. Require distinction between demand-pull, cost-push, and built-in inflation, plus monetary supply arguments.</thinking> <output><role>Macroeconomist</role><task>Explain the primary drivers of systemic inflation.</task><instructions>1. Distinguish between demand-pull inflation and cost-push inflation. 2. Explain the monetarist view (money supply growth) and the Keynesian view. 3. Detail how inflation expectations create built-in inflation. 4. Reference historical examples for each type.</instructions><output_format>Macroeconomic analysis outline.</output_format><constraints>Represent consensus views from both monetarist and Keynesian perspectives neutrally.</constraints></output>"
    ),
    (
        "difference between TCP and UDP",
        "accurate",
        "<thinking>A networking staple. Requires comparison across packet delivery guarantees, connection state, headers, speed, and real-world protocol examples.</thinking> <output><role>Network Architect</role><task>Compare TCP and UDP protocols.</task><instructions>1. Analyze differences in connection establishment (handshakes). 2. Compare ordering, reliability, and error checking mechanisms. 3. Highlight differences in header size. 4. Provide specific application layer protocol examples (e.g., HTTP vs DNS).</instructions><output_format>Comparative analysis with a summary table.</output_format><constraints>Avoid hand-waving explanations. Detail the packet validation differences.</constraints></output>"
    ),
    (
        "how does photosynthesis work",
        "accurate",
        "<thinking>Biochemical process. Needs light-dependent reactions (photosystem II & I, photolysis) and light-independent reactions (Calvin cycle) explained accurately.</thinking> <output><role>Plant Biologist</role><task>Explain the chemical stages of photosynthesis.</task><instructions>1. State the overall chemical equation. 2. Detail the light-dependent reactions occurring in the thylakoid membrane. 3. Detail the light-independent reactions (Calvin Cycle) in the stroma. 4. Explain the roles of chlorophyll, water splitting, ATP, and NADPH.</instructions><output_format>Biochemical review with step-by-step stages.</output_format><constraints>Use correct botanical and chemical terminology. Do not oversimplify the Calvin cycle.</constraints></output>"
    ),
    (
        "explain quantum superposition",
        "accurate",
        "<thinking>Requires mathematically grounded physical explanation without relying too heavily on pop-science analogies like cats that confuse the actual wave function mechanics.</thinking> <output><role>Quantum Physicist</role><task>Explain quantum superposition and the measurement problem.</task><instructions>1. Define superposition mathematically as a linear combination of state vectors. 2. Explain how the Schrödinger equation governs evolution until measurement. 3. Discuss the Copenhagen interpretation (wave function collapse) vs alternative interpretations. 4. Avoid misleading pop-science analogies.</instructions><output_format>Rigorous physics primer.</output_format><constraints>Maintain scientific rigor. Be clear about what wave function collapse represents physically.</constraints></output>"
    ),
    (
        "how does a catalytic converter work",
        "accurate",
        "<thinking>Requires explanation of reduction/oxidation catalysts, chemical reactions converting NOx, CO, and unburnt hydrocarbons to harmless gases.</thinking> <output><role>Chemical Engineer</role><task>Explain the chemical reactions within a catalytic converter.</task><instructions>1. Describe the structure (ceramic honeycomb coated with platinum/palladium/rhodium). 2. Explain the reduction catalyst phase (reducing NOx). 3. Explain the oxidation catalyst phase (oxidizing CO and hydrocarbons). 4. State the balance chemical equations for these transformations.</instructions><output_format>Technical engineering note with chemical equations.</output_format><constraints>All chemical equations must be balanced and accurate.</constraints></output>"
    ),
    (
        "explain blockchain consensus",
        "accurate",
        "<thinking>Explain PoW vs PoS. Must detail cryptographic hashing, difficulty adjustment, stake locking, slashing, and the Byzantine Generals Problem context.</thinking> <output><role>Cryptographic Protocol Researcher</role><task>Explain consensus mechanisms in blockchain networks.</task><instructions>1. Define the Byzantine Generals Problem in distributed computing. 2. Explain Proof of Work (mining, hash puzzles, energy usage). 3. Explain Proof of Stake (validators, collateral, slashing). 4. Compare their security assumptions and throughput limits.</instructions><output_format>Distributed systems comparative study.</output_format><constraints>Do not promote specific tokens. Keep the analysis strictly technical and architectural.</constraints></output>"
    ),
    (
        "what is a black hole event horizon",
        "accurate",
        "<thinking>Explain general relativity concept. Escape velocity exceeding speed of light, Schwarzschild radius calculation, time dilation effects.</thinking> <output><role>Astrophysicist</role><task>Explain the event horizon of a Schwarzschild black hole.</task><instructions>1. Define the event horizon in terms of space-time curvature and escape velocity. 2. Explain the Schwarzschild radius and how it is calculated from mass. 3. Describe the experience of an observer crossing the horizon vs an outside observer (gravitational time dilation). 4. Avoid sci-fi speculation.</instructions><output_format>Relativistic physics report.</output_format><constraints>Stay within general relativity framework. Clearly separate verified theory from quantum gravity hypotheses.</constraints></output>"
    ),
    (
        "how does standard deviation work",
        "accurate",
        "<thinking>Mathematical/statistical concept. Explain calculation steps (mean, deviations, squaring, mean of deviations, square root), variance connection, and standard distribution interpretation.</thinking> <output><role>Statistician</role><task>Explain the mathematical definition and calculation of standard deviation.</task><instructions>1. State the formula for both population and sample standard deviation, explaining the Bessel correction (n-1). 2. Break down the calculation into step-by-step algebra. 3. Explain what the value represents regarding data dispersion and the empirical rule (68-95-99.7).</instructions><output_format>Mathematical lesson with step-by-step calculation example.</output_format><constraints>Ensure mathematical formulas are formatted clearly (markdown or LaTeX).</constraints></output>"
    ),
    (
        "explain the water cycle steps",
        "accurate",
        "<thinking>Hydrological cycle. Need correct scientific terms: evaporation, transpiration, condensation, precipitation, infiltration, runoff, and groundwater storage.</thinking> <output><role>Hydrologist</role><task>Explain the earth's hydrological cycle.</task><instructions>1. Explain the driving force of the cycle (solar radiation, gravity). 2. Detail the primary phases: evaporation/transpiration, condensation, precipitation, and collection/infiltration. 3. Discuss the global mass balance of water conservation.</instructions><output_format>Earth science review.</output_format><constraints>Do not write for children. Use proper geological and environmental terms.</constraints></output>"
    ),
    (
        "what is the gold standard",
        "accurate",
        "<thinking>Economic history. Explain how gold exchange standard worked, fixed exchange rates, balance of payments adjustment, and transition to fiat currency.</thinking> <output><role>Financial Historian</role><task>Explain the mechanics and historical collapse of the Gold Standard.</task><instructions>1. Define the gold standard and the price-specie flow mechanism. 2. Explain how exchange rates were stabilized. 3. Detail the systemic vulnerabilities that led to its abandonment in the 20th century. 4. Compare it to the current fiat system.</instructions><output_format>Economic history briefing.</output_format><constraints>Ensure historical dates (e.g., Nixon shock, Bretton Woods) are accurate.</constraints></output>"
    ),
    (
        "difference between DNA and RNA",
        "accurate",
        "<thinking>Genetics. Structural differences (double helix vs single strand, deoxyribose vs ribose, thymine vs uracil) and functional differences (information storage vs coding/catalysis).</thinking> <output><role>Molecular Biologist</role><task>Compare DNA and RNA structurally and functionally.</task><instructions>1. Analyze sugar-phosphate backbone differences. 2. Compare nitrogenous bases. 3. Discuss stability differences and physical structure. 4. Detail their distinct cellular roles in transcription, translation, and replication.</instructions><output_format>Genetics comparison report with a detailed table.</output_format><constraints>Maintain strict biochemical naming conventions.</constraints></output>"
    ),
    (
        "how do optical fibers transmit data",
        "accurate",
        "<thinking>Physics of wave propagation. Total internal reflection, cladding vs core refractive indexes, dispersion, and signal degradation.</thinking> <output><role>Telecommunications Engineer</role><task>Explain data transmission through fiber optic cables.</task><instructions>1. Explain the physics principle of Total Internal Reflection. 2. Detail the difference in refractive index between the core and cladding. 3. Discuss modes of propagation (single-mode vs multi-mode). 4. Explain causes of attenuation and dispersion.</instructions><output_format>Physics of engineering report.</output_format><constraints>Do not omit the mathematical relationship of refractive index (Snell's Law).</constraints></output>"
    ),
    (
        "explain the role of the pancreas",
        "accurate",
        "<thinking>Human anatomy and physiology. Dual gland role: endocrine (insulin, glucagon from islets of Langerhans) and exocrine (digestive enzymes via pancreatic duct).</thinking> <output><role>Physiologist & Endocrinologist</role><task>Explain the exocrine and endocrine functions of the pancreas.</task><instructions>1. Detail the exocrine role (production of amylase, lipase, proteases, bicarbonate). 2. Detail the endocrine role (insulin, glucagon, somatostatin secretion). 3. Explain how these feedback loops maintain blood glucose homeostasis.</instructions><output_format>Medical physiology summary.</output_format><constraints>Use precise anatomical names (e.g., acinar cells, beta cells).</constraints></output>"
    ),
    (
        "what is the butterfly effect",
        "accurate",
        "<thinking>Chaos theory concept. Sensitive dependence on initial conditions in non-linear dynamical systems, Lorenz attractor history, predictability limits.</thinking> <output><role>Applied Mathematician</role><task>Explain the mathematical concept of Chaos Theory and the Butterfly Effect.</task><instructions>1. Define sensitive dependence on initial conditions. 2. Explain why non-linear differential equations are deterministic yet unpredictable. 3. Reference Edward Lorenz's weather prediction models. 4. Explain the difference between chaos and randomness.</instructions><output_format>Chaos theory primer.</output_format><constraints>Avoid treating it as a literal claim about butterflies. Explain the mathematics of error amplification.</constraints></output>"
    ),
    (
        "how do vaccines work",
        "accurate",
        "<thinking>Immunology. Antigen introduction, APC activation, T-cell helper interaction, B-cell antibody production, memory cell creation. Compare types (mRNA, viral vector, inactivated).</thinking> <output><role>Immunologist</role><task>Explain the physiological mechanism of vaccine-induced immunity.</task><instructions>1. Describe the primary immune response to an introduced antigen. 2. Explain how antigen-presenting cells trigger T and B lymphocytes. 3. Detail the generation of memory B and T cells. 4. Compare mechanisms of mRNA vaccines with traditional subunit/inactivated vaccines.</instructions><output_format>Immunology briefing with step-by-step biological processes.</output_format><constraints>Ensure medical terminology is accurate. Do not oversimplify cellular interactions.</constraints></output>"
    ),
    (
        "explain tectonic plate boundaries",
        "accurate",
        "<thinking>Geology. Convergent, divergent, transform boundaries. Associated geological formations (trenches, ridges, faults) and seismic activity.</thinking> <output><role>Geophysicist</role><task>Classify and explain the three types of tectonic plate boundaries.</task><instructions>1. Describe Divergent boundaries (seafloor spreading, rifts). 2. Describe Convergent boundaries (subduction zones, mountain building). 3. Describe Transform boundaries (strike-slip faults). 4. Associate specific geographical examples (e.g., Mid-Atlantic Ridge, San Andreas Fault) with each.</instructions><output_format>Geological classification framework.</output_format><constraints>Use correct geophysical terminology (lithosphere, asthenosphere, subduction).</constraints></output>"
    ),
    (
        "what is absolute zero",
        "accurate",
        "<thinking>Thermodynamics. Limit of classical thermodynamic system, zero kinetic energy, Kelvin scale definition, quantum ground state zero-point energy.</thinking> <output><role>Thermodynamics Physicist</role><task>Define and explain the physical limits of Absolute Zero.</task><instructions>1. State the value in Celsius and Kelvin. 2. Explain it classically as the minimization of translational kinetic energy. 3. Explain it via quantum mechanics (zero-point energy cannot be removed due to uncertainty principle). 4. Reference the third law of thermodynamics.</instructions><output_format>Physics lesson outline.</output_format><constraints>Do not claim that atoms stop moving entirely; explain the quantum zero-point energy restriction.</constraints></output>"
    ),
    (
        "explain inflation vs deflation",
        "accurate",
        "<thinking>Macroeconomics. Purchasing power effects, velocity of money, central bank policy tools, debt implications, deflationary spiral mechanism.</thinking> <output><role>Monetary Economist</role><task>Analyze the macroeconomic trade-offs between inflation and deflation.</task><instructions>1. Define both terms in relation to purchasing power. 2. Explain the mechanisms of a deflationary spiral (deferred consumption, rising real debt values). 3. Explain why central banks target a low positive inflation rate (e.g., 2%). 4. Compare policy tools used to combat both states.</instructions><output_format>Comparative economic report with a policy matrix.</output_format><constraints>Maintain policy-neutral analysis. Base arguments on empirical central bank theory.</constraints></output>"
    ),
    (
        "how does an airplane wing create lift",
        "accurate",
        "<thinking>Fluid dynamics. Newton's third law (action-reaction, air deflection) combined with Bernoulli's principle (pressure differences). Avoid the common 'equal transit time' fallacy.</thinking> <output><role>Aerodynamicist</role><task>Explain the physics of aerodynamic lift generation.</task><instructions>1. Address the 'equal transit time' fallacy and correct it. 2. Explain Bernoulli's principle in relation to flow velocity and pressure differences. 3. Explain Newton's third law (downwash, momentum conservation). 4. Discuss the integration of pressure distribution and deflection.</instructions><output_format>Aerodynamic reference sheet.</output_format><constraints>Do not use the equal transit time explanation under any circumstances. It is physically incorrect.</constraints></output>"
    ),
    (
        "what is the greenhouse effect",
        "accurate",
        "<thinking>Atmospheric physics. Absorption of solar radiation (UV/visible) by Earth, re-emission as infrared radiation, absorption and re-radiation by greenhouse gas molecules.</thinking> <output><role>Climate Scientist</role><task>Explain the physics of the greenhouse effect.</task><instructions>1. Differentiate incoming solar radiation wavelengths from outgoing earth radiation. 2. Explain the molecular absorption properties of CO2, CH4, and H2O (rotational/vibrational energy states). 3. Describe radiative forcing and thermal equilibrium. 4. Keep explanations grounded in thermodynamics.</instructions><output_format>Atmospheric physics paper structure.</output_format><constraints>Maintain thermodynamic rigor. Focus on radiative balance physics.</constraints></output>"
    ),
    # Additional accurate
    ("explain carbon offset credits", "accurate", "<thinking>Environmental economics. Additionality, double-counting risk, verification standards.</thinking> <output><role>Carbon Analyst</role><task>Explain carbon offsets.</task><instructions>1. Define carbon equivalence. 2. Detail additionality criteria. 3. Explain verified registries. 4. State project types (forestry vs technology).</instructions><output_format>Briefing note.</output_format><constraints>Discuss risks of greenwashing neutrally.</constraints></output>"),
    ("how does a refrigerator cooling system work", "accurate", "<thinking>Thermodynamics of vapor-compression refrigeration. Compressor, condenser, expansion valve, evaporator.</thinking> <output><role>Thermal Engineer</role><task>Explain refrigerator cycle.</task><instructions>1. Describe the refrigerant transformation states. 2. Walk through the four key thermal components. 3. State energy conservation laws applied.</instructions><output_format>Step-by-step mechanics explanation.</output_format><constraints>Strictly use correct phase change terms (condensation, evaporation).</constraints></output>"),
    ("what is game theory Nash equilibrium", "accurate", "<thinking>Game theory definition. Strategy set where no player has incentive to unilaterally deviate.</thinking> <output><role>Economics Professor</role><task>Explain Nash Equilibrium.</task><instructions>1. Define the mathematical concept. 2. Provide coordination game example. 3. Discuss multiple equilibria cases.</instructions><output_format>Theoretical brief.</output_format><constraints>Ensure game theory jargon is defined.</constraints></output>"),
    ("explain deflation economic consequences", "accurate", "<thinking>Deflationary trap mechanism. Wage rigidity, rising debt load, consumption deferral.</thinking> <output><role>Macroeconomist</role><task>Explain economic effects of deflation.</task><instructions>1. Analyze purchasing power changes. 2. Describe the deflationary spiral. 3. State central bank reaction tools.</instructions><output_format>Policy memo.</output_format><constraints>Focus on systemic liquidity effects.</constraints></output>"),
    ("difference between HTTP and HTTPS", "accurate", "<thinking>Network security. SSL/TLS handshakes, encryption layer, port allocations (80 vs 443).</thinking> <output><role>Security Engineer</role><task>Compare HTTP and HTTPS.</task><instructions>1. Explain the role of public key infrastructure. 2. Describe the data encapsulation differences. 3. Detail the performance impact of encryption.</instructions><output_format>Technical review.</output_format><constraints>Ensure accurate explanation of certificates.</constraints></output>"),
    ("how does cellular respiration work", "accurate", "<thinking>Biochemistry. Glycolysis, Krebs cycle, Electron Transport Chain, ATP synthesis yields.</thinking> <output><role>Cellular Biologist</role><task>Explain cellular respiration stages.</task><instructions>1. State overall chemical equation. 2. Detail glycolysis in cytoplasm. 3. Detail citric acid cycle and oxidative phosphorylation in mitochondria.</instructions><output_format>Biochemical review.</output_format><constraints>Use correct metabolic intermediates (pyruvate, NADH, FADH2).</constraints></output>"),
    ("explain quantum tunneling", "accurate", "<thinking>Quantum mechanics. Wave-particle duality, wave function decay through barrier, transmission probability.</thinking> <output><role>Theoretical Physicist</role><task>Explain quantum tunneling.</task><instructions>1. Define the potential energy barrier. 2. Solve the wave equation inside the barrier. 3. Discuss applications (STM, nuclear fusion).</instructions><output_format>Theoretical note.</output_format><constraints>Include probability wave explanation.</constraints></output>"),
    ("how does an engine spark plug work", "accurate", "<thinking>Electrical mechanics. High voltage dielectric breakdown of air gap generating spark.</thinking> <output><role>Automotive Engineer</role><task>Explain spark plug ignition.</task><instructions>1. Detail structural layers (terminal, ceramic insulator, electrodes). 2. Explain ignition coil voltage multiplication. 3. Describe combustion propagation.</instructions><output_format>Engineering manual page.</output_format><constraints>Keep mechanics aligned with Otto cycle.</constraints></output>"),
    ("explain Raft consensus algorithm", "accurate", "<thinking>Distributed systems consensus. Leader election, log replication, safety invariants.</thinking> <output><role>Systems Architect</role><task>Explain Raft algorithm.</task><instructions>1. Define states (Leader, Follower, Candidate). 2. Detail heartbeat cycles. 3. Explain split vote resolution.</instructions><output_format>Technical specification summary.</output_format><constraints>Contrast with Paxos complexity where relevant.</constraints></output>"),
    ("what is dark matter evidence", "accurate", "<thinking>Astrophysics. Galactic rotation curves, gravitational lensing, cosmic microwave background power spectrum.</thinking> <output><role>Cosmologist</role><task>List dark matter evidence.</task><instructions>1. Explain Vera Rubin's rotation curve findings. 2. Describe gravitational lensing distortions. 3. Detail CMB fluctuations.</instructions><output_format>Scientific review.</output_format><constraints>Distinguish observation from theoretical models.</constraints></output>"),
    ("how does linear regression work", "accurate", "<thinking>Statistics/ML. Ordinary Least Squares (OLS), residual sum of squares minimization, coefficient estimation.</thinking> <output><role>Data Scientist</role><task>Explain linear regression model.</task><instructions>1. State the linear hypothesis equation. 2. Explain error terms and cost function. 3. Describe slope and intercept calculations.</instructions><output_format>Mathematical summary.</output_format><constraints>Use correct statistics notation.</constraints></output>"),
    ("explain carbon cycle steps", "accurate", "<thinking>Environmental cycles. Carbon sinks, photosynthesis, respiration, decomposition, fossilization, oceanic dissolution.</thinking> <output><role>Biogeochemist</role><task>Explain global carbon cycle.</task><instructions>1. Detail fast vs slow cycle components. 2. Identify major reservoir boundaries. 3. Outline anthropogenic impacts.</instructions><output_format>Scientific report.</output_format><constraints>Ensure accurate mass transfers descriptions.</constraints></output>"),
    ("what is fiat money system history", "accurate", "<thinking>Monetary history. Value by government decree, decoupling from reserves, trust mechanisms.</thinking> <output><role>Financial Historian</role><task>Explain fiat currency origins.</task><instructions>1. Transition from commodity standards to credit money. 2. Role of central banks. 3. Inflation risks in fractional reserve systems.</instructions><output_format>Historical review.</output_format><constraints>Keep historical sequence accurate.</constraints></output>"),
    ("difference between transcription and translation in genetics", "accurate", "<thinking>Biology. RNA Polymerase synthesis of mRNA vs Ribosome assembly of amino acid chain.</thinking> <output><role>Geneticist</role><task>Compare transcription and translation.</task><instructions>1. Detail cellular locations of both phases. 2. Define templates and enzymes utilized. 3. Explain start/stop signaling.</instructions><output_format>Comparative brief.</output_format><constraints>Maintain biochemical precision.</constraints></output>"),
    ("how do solar cells convert light to electricity", "accurate", "<thinking>Solid state physics. P-N junction, photovoltaic effect, electron-hole pair generation.</thinking> <output><role>Materials Scientist</role><task>Explain solar cell physics.</task><instructions>1. Explain silicon doping (N-type vs P-type). 2. Describe depletion zone formation. 3. Explain how incoming photons mobilize electrons.</instructions><output_format>Technical brief.</output_format><constraints>Focus on energy bandgap theory.</constraints></output>"),
    ("explain the role of the thyroid gland", "accurate", "<thinking>Anatomy/physiology. T3, T4, calcitonin secretion, regulation by TSH, basal metabolic rate.</thinking> <output><role>Endocrinologist</role><task>Explain thyroid physiology.</task><instructions>1. Detail synthesized hormones. 2. Describe hypothalamic-pituitary-thyroid axis. 3. Explain iodine dependency.</instructions><output_format>Medical reference sheet.</output_format><constraints>Keep clinical parameters accurate.</constraints></output>"),
    ("what is the doppler effect physics", "accurate", "<thinking>Wave mechanics. Apparent frequency shift due to relative velocity between source and observer.</thinking> <output><role>Acoustic Physicist</role><task>Explain Doppler Effect.</task><instructions>1. Write mathematical formula for moving source/observer. 2. Describe acoustic compression. 3. Explain astronomical redshift.</instructions><output_format>Academic primer.</output_format><constraints>Ensure formula variables are defined.</constraints></output>"),
    ("how do antibiotics work", "accurate", "<thinking>Pharmacology. Cell wall synthesis inhibition, protein synthesis inhibition, DNA replication interference.</thinking> <output><role>Microbiologist</role><task>Explain antibiotic target mechanisms.</task><instructions>1. Compare bactericidal vs bacteriostatic actions. 2. Detail targets (peptidoglycan wall, ribosomes). 3. Discuss resistance mechanics.</instructions><output_format>Pharmacological review.</output_format><constraints>Do not write for viral infections.</constraints></output>"),
    ("explain ocean currents thermohaline circulation", "accurate", "<thinking>Oceanography. Density-driven currents (temperature and salinity gradients), conveyor belt model.</thinking> <output><role>Oceanographer</role><task>Explain global conveyor belt.</task><instructions>1. Detail sinking zones in North Atlantic. 2. Explain salinity extraction by sea ice. 3. Discuss thermal stabilization impact on Europe.</instructions><output_format>Geophysical briefing.</output_format><constraints>Use proper density stratification terminology.</constraints></output>"),
    ("what is the triple point of water", "accurate", "<thinking>Thermodynamics. Temperature and pressure coordinate where solid, liquid, and gas coexist in thermodynamic equilibrium.</thinking> <output><role>Physical Chemist</role><task>Define triple point of water.</task><instructions>1. Give exact pressure and temperature values. 2. Explain Gibbs' Phase Rule application. 3. Discuss calibration use in Kelvin scale.</instructions><output_format>Thermodynamics note.</output_format><constraints>Pressure must be in Pascals or atmospheres.</constraints></output>"),
    ("explain fiscal vs monetary policy", "accurate", "<thinking>Economics. Government spending/taxation vs Central bank interest rates/money supply.</thinking> <output><role>Macroeconomist</role><task>Compare fiscal and monetary policy.</task><instructions>1. Define primary actors (Treasury vs Central Bank). 2. List tools (bonds, taxes, interest rates, QE). 3. Detail implementation lag difference.</instructions><output_format>Economic briefing paper.</output_format><constraints>Remain neutral regarding school of thought.</constraints></output>"),
    ("how does a gyroscope maintain orientation", "accurate", "<thinking>Classical mechanics. Conservation of angular momentum, precession torque.</thinking> <output><role>Mechanical Physicist</role><task>Explain gyroscopic stabilization.</task><instructions>1. State angular momentum formula. 2. Explain rigidity in space. 3. Define precession mechanics under force.</instructions><output_format>Physics lesson.</output_format><constraints>Ensure mathematical terms are defined.</constraints></output>"),
    ("what is the ozone layer depletion chemistry", "accurate", "<thinking>Atmospheric chemistry. CFC photolysis, chlorine radical catalysis cycle, polar stratospheric clouds.</thinking> <output><role>Atmospheric Chemist</role><task>Explain ozone depletion reactions.</task><instructions>1. Detail UV impact on CFC molecules. 2. Write the chlorine radical chain reaction equations. 3. State how one radical destroys thousands of ozone molecules.</instructions><output_format>Chemical analysis.</output_format><constraints>Equations must balance.</constraints></output>"),
    ("difference between mitosis and meiosis", "accurate", "<thinking>Cytology. Somatic division (diploid output) vs gamete division (haploid output, recombination).</thinking> <output><role>Cellular Biologist</role><task>Compare mitosis and meiosis.</task><instructions>1. Contrast final chromosome counts. 2. Describe homologous recombination in prophase I. 3. Detail number of cellular divisions.</instructions><output_format>Biomedical outline.</output_format><constraints>Maintain correct phase naming (metaphase, anaphase).</constraints></output>"),
    ("how does GPS triangulation work", "accurate", "<thinking>Satellite navigation. Trilateration (not triangulation), signal travel time, atomic clock sync, relativity correction.</thinking> <output><role>Geodesy Engineer</role><task>Explain GPS location tracking.</task><instructions>1. Differentiate trilateration from triangulation. 2. Describe intersection of spheres. 3. Explain the necessity of Einstein's relativity corrections for clocks.</instructions><output_format>Technical review.</output_format><constraints>Clearly explain general and special relativity effects on satellite time.</constraints></output>"),
    ("explain the difference between mass and weight", "accurate", "<thinking>Physics. Quantity of matter (inertia) vs gravitational force acting on that matter.</thinking> <output><role>Classical Physicist</role><task>Contrast mass and weight.</task><instructions>1. Compare units (Kg vs Newtons). 2. Explain measurement differences. 3. Describe outcome changes on another planet.</instructions><output_format>Academic brief.</output_format><constraints>Keep gravitational constant calculations accurate.</constraints></output>"),
    ("how does an MRI machine scan the body", "accurate", "<thinking>Medical physics. Proton alignment under strong magnetic field, RF pulse disturbance, relaxation time detection.</thinking> <output><role>Biomedical Engineer</role><task>Explain MRI physics.</task><instructions>1. Describe hydrogen proton behavior under magnetic fields. 2. Explain radiofrequency excitation. 3. Detail T1 and T2 relaxation signals translation into images.</instructions><output_format>Medical physics review.</output_format><constraints>Ensure focus on magnetic spin mechanics.</constraints></output>"),
    ("what is the first law of thermodynamics", "accurate", "<thinking>Thermodynamics. Conservation of energy: internal energy change equals heat added minus work done by system.</thinking> <output><role>Thermodynamics Specialist</role><task>State and derive the first law of thermodynamics.</task><instructions>1. Write formula: dU = dQ - dW. 2. Define closed system conditions. 3. Explain difference between state functions and path variables.</instructions><output_format>Physics study notes.</output_format><constraints>Ensure sign conventions are clearly stated.</constraints></output>"),
    ("explain key differences between bacteria and viruses", "accurate", "<thinking>Microbiology. Living cellular organism with autonomous metabolism vs non-cellular parasite needing host machinery.</thinking> <output><role>Microbiologist</role><task>Compare bacteria and viruses.</task><instructions>1. Contrast structural components (cell wall/ribosomes vs capsid/envelope). 2. Describe reproduction mechanics. 3. Compare treatment methods (antibiotics vs antivirals).</instructions><output_format>Comparative grid layout.</output_format><constraints>Keep biological definitions standard.</constraints></output>"),
    ("how do high and low pressure weather systems form", "accurate", "<thinking>Meteorology. Air column temperature variations, rising vs sinking air currents, Coriolis force rotations.</thinking> <output><role>Meteorologist</role><task>Explain weather pressure systems.</task><instructions>1. Describe sinking air creating high pressure (anticyclones). 2. Describe rising warm air creating low pressure (cyclones). 3. Detail rotation directions in Northern vs Southern Hemispheres.</instructions><output_format>Atmospheric briefing.</output_format><constraints>Link pressure changes to moisture condensation patterns.</constraints></output>"),

    # =========================================================================
    # CUSTOM (56 examples total)
    # =========================================================================
    (
        "generate user persona",
        "custom",
        "<thinking>User personas require specific formatting: demographics, goals, frustrations, tech stack, and a quote. I need to output structured XML templates for product designers.</thinking> <output><role>Lead UX Researcher</role><context>Designing a mobile B2B project management app for remote construction supervisors.</context><task>Create a comprehensive user persona.</task><instructions>1. Provide a realistic name, job title, and demographic profile. 2. Outline their daily workflow. 3. Define 3 primary goals and 3 key frustrations. 4. Detail their technology preferences and environment. 5. Include a representative quote summing up their challenge.</instructions><output_format>UX design template with sections: Profile, Quote, Goals, Pain Points, Tech Affinity.</output_format><constraints>Ensure the persona represents a realistic construction workspace, not a general office worker.</constraints></output>"
    ),
    (
        "create api error response schema",
        "custom",
        "<thinking>API error design needs standardization: RFC 7807 problem details is the modern standard. Build a template that enforces this structure.</thinking> <output><role>API Platform Architect</role><task>Design a standardized JSON error response schema.</task><instructions>1. Use RFC 7807 (Problem Details for HTTP APIs) structure. 2. Include fields: type, title, status, detail, instance, and an optional invalid_params list for validation errors. 3. Provide a JSON example for a validation error (HTTP 400) and an authorization error (HTTP 401).</instructions><output_format>Schema description followed by JSON code blocks.</output_format><constraints>All JSON payloads must validate against RFC 7807 schema rules.</constraints></output>"
    ),
    (
        "draft feature release email",
        "custom",
        "<thinking>Product announcement emails need a specific narrative arc: value prop, the feature, how it works, and a CTA, with clear email header metadata.</thinking> <output><role>Product Marketing Manager</role><context>Launching a new AI-powered document summarizer feature for an enterprise SaaS.</context><task>Draft the feature announcement email.</task><instructions>1. Provide Subject Line and Preview Text. 2. Structure the body: The Problem (document fatigue), The Solution (AI summarizer), Core Benefits, and a clear Call to Action (CTA). 3. Maintain an exciting yet professional B2B tone.</instructions><output_format>Email template layout with placeholder tags.</output_format><constraints>Do not exceed 300 words. Keep paragraphs under 3 sentences for readability.</constraints></output>"
    ),
    (
        "write interview questions for product manager",
        "custom",
        "<thinking>Product manager interviews evaluate multiple axes: execution, product sense, leadership, and technical collaboration. Create a structured evaluation rubric.</thinking> <output><role>VP of Product</role><task>Draft a structured interview question list for Senior PM candidates.</task><instructions>1. Organize questions by axis: Execution, Product Sense, Leadership. 2. Provide 2 questions per axis. 3. For each question, explain what a strong response looks like (evaluation rubric).</instructions><output_format>Interview guide with questions and evaluation guidelines.</output_format><constraints>Focus on behavior-based questions. Avoid generic puzzle questions.</constraints></output>"
    ),
    (
        "generate code review checklist",
        "custom",
        "<thinking>Code review checklists must be actionable and organized by priority (security, correctness, performance) rather than stylistic preferences that should be handled by linting.</thinking> <output><role>Engineering Manager</role><task>Create a team code review checklist.</task><instructions>1. Categorize checks into: Security (SQLi, auth), Correctness (edge cases, state), and Maintainability (naming, duplication). 2. Exclude checks that can be automated via linters. 3. Format as a markdown list with brief explanations of the risk for each point.</instructions><output_format>Markdown checklist.</output_format><constraints>Limit checklist to 10 high-impact items. Keep items objective.</constraints></output>"
    ),
    (
        "create bug report template",
        "custom",
        "<thinking>A high-quality bug report template contains specific markdown fields to ensure developers have reproduction steps, environment details, and logs.</thinking> <output><role>QA Lead</role><task>Design a GitHub issue template for reporting bugs.</task><instructions>Include sections for: Summary, Steps to Reproduce, Expected vs Actual Behavior, Environment (OS, browser, version), Logs/Screenshots placeholders, and Severity estimation.</instructions><output_format>GitHub Markdown issue template (raw markdown block).</output_format><constraints>Ensure fields force users to provide reproducible steps rather than vague descriptions.</constraints></output>"
    ),
    (
        "write standard operating procedure for deployment",
        "custom",
        "<thinking>An SOP for deployment needs chronological steps, pre-flight checks, rollback criteria, and communication channels.</thinking> <output><role>DevOps Lead</role><context>Deploying a microservice update to Kubernetes staging/production environments.</context><task>Write a Standard Operating Procedure (SOP) document.</task><instructions>1. Outline pre-deployment checklist (build validation, database migration backup). 2. Define the exact step-by-step deployment commands. 3. Define metrics to monitor post-deployment (error rates, latency). 4. State the explicit rollback trigger criteria and rollback execution steps.</instructions><output_format>Standard Technical SOP layout with numbered steps.</output_format><constraints>Do not gloss over the rollback procedure; it must be as detailed as the deploy procedure.</constraints></output>"
    ),
    (
        "draft privacy policy clause for cookies",
        "custom",
        "<thinking>Cookie consent clauses must comply with GDPR and CCPA, detailing what is collected, why, and how user can opt out.</thinking> <output><role>Legal Counsel (Data Privacy Specialist)</role><task>Draft a website cookie policy clause.</task><instructions>1. Explain the use of cookies (essential, analytical, marketing). 2. Explain how third-party cookies are managed. 3. Detail the opt-out/consent management process. 4. Maintain formal, legally-binding prose.</instructions><output_format>Legal policy document section.</output_format><constraints>Must comply with GDPR Article 7 requirements for active consent explanation.</constraints></output>"
    ),
    (
        "create a weekly team sync agenda",
        "custom",
        "<thinking>Team sync agendas should prevent long updates and prioritize unblocking team members and mapping sprint progress. Timeboxing is critical.</thinking> <output><role>Agile Scrum Master</role><context>A distributed team of 8 software engineers running 2-week sprints.</context><task>Draft a 30-minute weekly sync agenda.</task><instructions>1. Timebox every segment (e.g., Wins: 5m, Blockers: 15m, Backlog Refinement: 10m). 2. Provide guidelines for meeting preparation. 3. Outline rules for keeping the meeting concise (e.g., take deep dives offline).</instructions><output_format>Meeting agenda template with time allocations.</output_format><constraints>Total time must equal exactly 30 minutes. Focus on blocker resolution.</constraints></output>"
    ),
    (
        "write post-mortem incident report outline",
        "custom",
        "<thinking>Incident post-mortems require a blame-free analysis focusing on timeline, root cause, detection, and action items to prevent recurrence.</thinking> <output><role>Site Reliability Engineer (SRE)</role><task>Design a Blameless Post-Mortem incident report template.</task><instructions>Include sections for: Incident Metadata (severity, owners, timing), Timeline (detection to recovery), Root Cause Analysis (using 5 Whys), Impact summary (users affected, database downtime), and Action Items (preventative work categorized by priority).</instructions><output_format>Markdown template with placeholder instructions.</output_format><constraints>Enforce a blameless tone. The document must focus on systemic faults, not human errors.</constraints></output>"
    ),
    (
        "generate user story with acceptance criteria",
        "custom",
        "<thinking>User stories need the standard Connextra format (As a, I want, So that) and Gherkin syntax (Given, When, Then) for acceptance criteria.</thinking> <output><role>Technical Product Owner</role><context>Adding a 'save search query' feature for an e-commerce platform.</context><task>Draft a user story with detailed acceptance criteria.</task><instructions>1. Write the core user story in As/I/So format. 2. Write 3 acceptance scenarios using Gherkin syntax (Given/When/Then) covering: happy path, query limit reached, and deleting a saved query.</instructions><output_format>Product backlog item format.</output_format><constraints>Acceptance criteria must be deterministic and testable.</constraints></output>"
    ),
    (
        "write cold email template for sales",
        "custom",
        "<thinking>Cold emails need personalization, low friction, a clear value proposition, and a low-commitment call to action.</thinking> <output><role>B2B Sales Copywriter</role><context>Targeting CTOs for a serverless performance monitoring tool.</context><task>Draft a cold outreach email.</task><instructions>1. Write 3 alternative attention-grabbing subject lines. 2. Write a short body (under 150 words) establishing the pain point (unpredictable server costs), our solution, and a low-friction CTA (e.g., 'Do you have 5 minutes next Thursday?').</instructions><output_format>Sales template with personalization brackets.</output_format><constraints>No pushy or generic corporate jargon. The focus must be on cost savings and developer experience.</constraints></output>"
    ),
    (
        "create onboarding checklist for developers",
        "custom",
        "<thinking>Developer onboarding checklists should cover local environment setup, permission provisioning, documentation reading, and early milestones.</thinking> <output><role>Principal Engineer</role><task>Design a 30-day developer onboarding checklist.</task><instructions>1. Divide checklist into: Day 1 (setup & accounts), Week 1 (first PR to staging), and Day 30 (independent feature deployment). 2. Provide links/placeholders for local repository setup guides and architecture docs.</instructions><output_format>Chronological markdown checklist.</output_format><constraints>Make milestones realistic. Prioritize early system understanding over speed of delivery.</constraints></output>"
    ),
    (
        "generate release notes markdown",
        "custom",
        "<thinking>Release notes should separate new features, bug fixes, and security patches, explaining user value rather than just raw git logs.</thinking> <output><role>Technical Writer</role><context>V2.4.0 release of a database management tool.</context><task>Draft the release notes markdown.</task><instructions>1. Organize by: Features, Improvements, Bug Fixes, and Security. 2. Write a brief description of the benefit for each item. 3. Include a migration notice if there are breaking API changes.</instructions><output_format>Markdown documentation.</output_format><constraints>Do not copy-paste git commits. Translate technical commits to user value.</constraints></output>"
    ),
    (
        "write pull request description template",
        "custom",
        "<thinking>PR templates should collect changes, testing verification, related issues, and deployment considerations from developers.</thinking> <output><role>Lead Developer</role><task>Design a GitHub pull request template.</task><instructions>Include markdown sections for: Type of change (dropdown/checkbox), Description of changes, How this was tested (unit test coverage, manual steps), Related Jira tickets/Issues, and Deployment notes (migrations, config changes).</instructions><output_format>Pull request description markdown template.</output_format><constraints>Force developers to document their testing process to reduce bugs.</constraints></output>"
    ),
    (
        "create database indexing guidelines",
        "custom",
        "<thinking>Indexing guidelines need to specify when to use indexes (FKs, search columns) and when to avoid them (high-write tables, low-cardinality columns) to prevent write latency.</thinking> <output><role>Database Administrator</role><task>Create a database indexing guidelines document for software engineers.</task><instructions>1. Define rules for primary key and foreign key indexing. 2. Explain how compound indexes work and the left-prefix rule. 3. Provide anti-patterns (e.g., indexing boolean columns, over-indexing). 4. Format as a technical style guide.</instructions><output_format>Markdown database guidelines.</output_format><constraints>Keep rules general to SQL databases (PostgreSQL/MySQL), highlighting index overhead on writes.</constraints></output>"
    ),
    (
        "draft system architecture review template",
        "custom",
        "<thinking>System architecture reviews should assess scaling, security, data flow, failure modes, and operational costs.</thinking> <output><role>Solutions Architect</role><task>Design a System Architecture Review (SAR) template.</task><instructions>Create sections for: Executive Summary, System Components & Data Flow, Scalability & Performance Strategy, Security & Compliance, Fault Tolerance & Disaster Recovery, and Cost Estimation.</instructions><output_format>Structured technical architecture template.</output_format><constraints>All sections must guide the writer to identify trade-offs (e.g., latency vs consistency).</constraints></output>"
    ),
    (
        "generate QA test execution report structure",
        "custom",
        "<thinking>QA execution reports need summary metrics (pass, fail, block rate), links to test runs, list of defects found, and go/no-go recommendation.</thinking> <output><role>QA Manager</role><task>Design a Test Execution Report template.</task><instructions>Include markdown tables for test run statistics, lists for open critical defects, details about test environment, and a clear 'Release Readiness' recommendation section.</instructions><output_format>QA reporting template with mock data illustration.</output_format><constraints>Focus on metrics that determine release risk, not just pass/fail counts.</constraints></output>"
    ),
    (
        "write security audit findings template",
        "custom",
        "<thinking>Security findings must report vulnerability name, CVSS score, description, proof of concept, risk impact, and remediation steps.</thinking> <output><role>Cybersecurity Auditor</role><task>Design a security vulnerability report template.</task><instructions>Include fields for: Finding Title, Severity (Critical/High/Medium/Low), CVSS vector, Affected component, Vulnerability Description, Proof of Concept (PoC) code block, Business Risk Impact, and Remediation Steps.</instructions><output_format>Audit findings section template.</output_format><constraints>Remediation steps must prioritize root-cause code fixes over perimeter mitigations.</constraints></output>"
    ),
    (
        "create technical debt assessment log",
        "custom",
        "<thinking>Technical debt logs must track debt items, code location, complexity, estimated impact on velocity, and estimated refactoring effort.</thinking> <output><role>Software Architect</role><task>Design a Technical Debt Log template.</task><instructions>Define a markdown table format with columns: ID, Component, Description of Debt, Estimated Impact (High/Medium/Low), Refactor Effort (Sprint points), and Priority ranking.</instructions><output_format>Markdown template with tracking table.</output_format><constraints>Provide instructions on how to calculate the 'velocity impact' objectively.</constraints></output>"
    ),
    (
        "draft vendor evaluation matrix scorecard",
        "custom",
        "<thinking>Vendor selection scorecards need weightings, scores across security, price, SLA, features, and an overall scoring formula.</thinking> <output><role>IT Procurement Specialist</role><task>Create a vendor evaluation scorecard template.</task><instructions>1. Set categories: Compliance/Security, Features, SLA/Support, Pricing. 2. Assign weights to each category (totaling 100%). 3. Include scoring columns (1-5 scale) and calculated weighted score columns.</instructions><output_format>Markdown table scorecard.</output_format><constraints>The weight calculation formulas must be explicitly explained in text.</constraints></output>"
    ),
    (
        "generate API deprecation notice email",
        "custom",
        "<thinking>API deprecation notices require advance warnings, migration path examples, sunset dates, and support links.</thinking> <output><role>Developer Relations Engineer</role><context>Deprecating V1 payment endpoints in favor of V2 REST paths.</context><task>Write a developer email notification for API deprecation.</task><instructions>1. State the exact Sunset Date clearly. 2. Provide a migration guide snippet showing V1 vs V2 JSON payload comparison. 3. Detail the backward-compatibility support plan. 4. Maintain a clear, helpful developer-focused tone.</instructions><output_format>Email template layout.</output_format><constraints>Do not hide the timeline. The sunset date must be highlighted near the top.</constraints></output>"
    ),
    (
        "write git commit message guide",
        "custom",
        "<thinking>Enforce Conventional Commits standard (type(scope): subject, body, footer) for clean git logs and automated versioning.</thinking> <output><role>DevOps Specialist</role><task>Create a Conventional Commits reference guide.</task><instructions>1. Explain the format: type(scope): subject. 2. Define valid types: feat, fix, chore, refactor, test, docs, style. 3. Provide examples of good vs bad commit messages. 4. Explain how breaking changes are marked in the footer.</instructions><output_format>Developer cheat-sheet markdown guide.</output_format><constraints>Align strictly with the Conventional Commits 1.0.0 specification.</constraints></output>"
    ),
    (
        "create a technical writing style guide checklist",
        "custom",
        "<thinking>Technical writing checklists should focus on active voice, sentence length, terminology consistency, formatting, and reader-first tone.</thinking> <output><role>Principal Technical Writer</role><task>Draft a technical style checklist for engineers writing documentation.</task><instructions>Create checks for: Sentence structure (prefer active voice, limit length), Formatting (correct code block syntax), Terminology (standardize API terms), and Accessibility (alt text for diagrams).</instructions><output_format>Markdown checklist.</output_format><constraints>Limit rules to high-impact improvements. Focus on readability and translation friendliness.</constraints></output>"
    ),
    (
        "generate product roadmap timeline format",
        "custom",
        "<thinking>Roadmaps should group initiatives by theme (e.g., Growth, Infrastructure) and time horizons (Now, Next, Later) rather than specific dates to prevent project management stress.</thinking> <output><role>Director of Product Management</role><task>Design a high-level strategic roadmap format.</task><instructions>Create a matrix format showing Now, Next, and Later horizons across Product Growth, System Stability, and Developer Tooling pillars. Explain how to prioritize items into each column.</instructions><output_format>Roadmap framework table.</output_format><constraints>Avoid specific calendar dates; focus entirely on strategic horizons.</constraints></output>"
    ),
    (
        "generate release checklist",
        "custom",
        "<thinking>The user needs a checklist for software releases. I should design a custom release checklist XML template detailing pre-release validation, release gates, smoke tests, and rollback criteria.</thinking> <output><role>Release Manager</role><task>Draft a release readiness checklist.</task><instructions>Provide steps for verification of dependencies, database schema checks, smoke testing protocols, and rollback approvals.</instructions><output_format>Markdown checklist layout with owners and status columns.</output_format><constraints>Ensure all steps are actionable and state clear success criteria.</constraints></output>"
    ),
    # Additional custom
    ("generate customer feedback survey template", "custom", "<thinking>Survey templates require structured questionnaire design targeting usability, support, and product speed.</thinking> <output><role>UX Researcher</role><task>Draft NPS/CSAT survey structure.</task><instructions>Provide 5 time-boxed feedback questions using Likert scales and open-ended text fields.</instructions><output_format>Feedback matrix template.</output_format><constraints>Survey must be designed to be completed under 2 minutes.</constraints></output>"),
    ("create JSON schema for user profile API", "custom", "<thinking>JSON schema validation rules for user resource. Need data types, constraints, formats.</thinking> <output><role>Backend Architect</role><task>Generate JSON schema.</task><instructions>Provide draft schema for properties: username, email (format: email), preferences, and metadata fields.</instructions><output_format>JSON schema block.</output_format><constraints>Ensure valid JSON schema draft-07 syntax.</constraints></output>"),
    ("draft company milestone celebration email", "custom", "<thinking>Internal communications milestone. Tone should celebrate team effort, list achievements, state future direction.</thinking> <output><role>CEO Communications Advisor</role><task>Draft internal milestone email.</task><instructions>Write a message celebrating 100k users. Emphasize team contribution and upcoming sprint challenges.</instructions><output_format>Email template block.</output_format><constraints>Keep focus on collective effort, avoid individual callouts.</constraints></output>"),
    ("write interview questions for DevOps engineer", "custom", "<thinking>DevOps interview questions focusing on infrastructure as code, orchestration, incident management.</thinking> <output><role>Principal DevOps Engineer</role><task>Create DevOps interview questions.</task><instructions>Draft 4 scenario-based questions evaluating disaster recovery and Terraform states.</instructions><output_format>Interview rubric guide.</output_format><constraints>Prioritize hands-on troubleshooting questions over trivia.</constraints></output>"),
    ("generate code review template for database migrations", "custom", "<thinking>Database migration code review. Focus on locking behaviors, column index rebuilds, zero-downtime guidelines.</thinking> <output><role>DBA & Dev Lead</role><task>Create migration review guide.</task><instructions>Provide checks for lock timeouts, backward compatibility, and rollback verification scripts.</instructions><output_format>Markdown checklist.</output_format><constraints>Enforce zero-downtime rules for schema alterations.</constraints></output>"),
    ("create feature request issue template", "custom", "<thinking>Feature request template. Need fields: User Problem, Proposed Solution, Alternatives Considered, Context/Value.</thinking> <output><role>Product Manager</role><task>Create issue template.</task><instructions>Draft issue fields that guide users to explain the underlying problem instead of just listing solutions.</instructions><output_format>Markdown issue form.</output_format><constraints>Keep instructions inside templates clear.</constraints></output>"),
    ("write procedure for handling production database restore", "custom", "<thinking>SOP for database restore. Critical sequence: backup active logs, verify sandbox environment, execute restore, validation checks.</thinking> <output><role>SRE Lead</role><task>Create DB restore checklist.</task><instructions>Detail steps to download backup, verify signature, restore to staging, run sanity queries, redirect traffic.</instructions><output_format>Technical SOP block.</output_format><constraints>Emphasize verification in staging before touching production paths.</constraints></output>"),
    ("draft GDPR data deletion request policy clause", "custom", "<thinking>GDPR Article 17 right to erasure. Explain user rights, timing (30 days), verification requirements.</thinking> <output><role>Privacy Lawyer</role><task>Draft privacy clause.</task><instructions>Write a clear description of the data deletion intake workflow and deletion validation steps.</instructions><output_format>Legal policy section.</output_format><constraints>Tone must be formal and compliant with EU guidelines.</constraints></output>"),
    ("create a quarterly project planning session agenda", "custom", "<thinking>Quarterly project planning. Enforce alignment on goals, capacity check, roadmapping, and priority locks.</thinking> <output><role>Agile Coach</role><task>Draft quarterly planning agenda.</task><instructions>Provide timeline structure (total 4 hours) split into retrospective, capacity calculation, mapping, and consensus voting.</instructions><output_format>Agenda template.</output_format><constraints>Detail time limits for each segment strictly.</constraints></output>"),
    ("write post-mortem documentation for system outage", "custom", "<thinking>SRE incident review. Timeline, impact metrics, root cause, short-term fixes, long-term preventions.</thinking> <output><role>Site Reliability Engineer</role><task>Draft post-mortem template.</task><instructions>Provide empty fields with instructions on documenting root causes via 5-whys technique.</instructions><output_format>Markdown template.</output_format><constraints>Strictly forbid blameworthy descriptions of engineer errors.</constraints></output>"),
    ("generate user story for authentication with OAuth2", "custom", "<thinking>User story for OAuth integration. Standard structure, Gherkin criteria for third-party scopes and token expiry.</thinking> <output><role>Product Owner</role><task>Write OAuth user story.</task><instructions>Provide story for Google/GitHub sign-in. Include scenarios for user rejection and token refresh failures.</instructions><output_format>Jira ticket markdown.</output_format><constraints>Gherkin syntax (Given/When/Then) must be correct.</constraints></output>"),
    ("write cold LinkedIn outreach message for sales", "custom", "<thinking>LinkedIn outreach. Concise (under 300 characters), value-focused, low-friction reply request.</thinking> <output><role>B2B Copywriter</role><task>Draft LinkedIn sales message.</task><instructions>Provide 2 options highlighting infrastructure monitoring integration benefits.</instructions><output_format>Message templates.</output_format><constraints>Keep copy strictly under LinkedIn character limit.</constraints></output>"),
    ("create onboarding tasks list for junior developers", "custom", "<thinking>Junior onboarding. Focus on sandboxed local setup, understanding Git branching, minor documentation edits first week.</thinking> <output><role>Engineering Mentor</role><task>Draft junior onboarding plan.</task><instructions>Structure day 1 to 5 steps focusing on development environment configuration and local testing.</instructions><output_format>Markdown checklist.</output_format><constraints>Milestones should have no production write permissions in week 1.</constraints></output>"),
    ("generate release notes template", "custom", "<thinking>Release notes format. Focus on highlights, breaking changes, contributor credits, upgrading steps.</thinking> <output><role>Technical Writer</role><task>Design release notes template.</task><instructions>Draft structured sections for library consumers with upgrading syntax placeholders.</instructions><output_format>Markdown template.</output_format><constraints>Keep upgrading sections highly visible.</constraints></output>"),
    ("write pull request guidelines markdown", "custom", "<thinking>PR guidelines for open source project. Enforce branch name styling, test coverage requirements, documentation updates.</thinking> <output><role>OS Maintainer</role><task>Write PR guidelines.</task><instructions>Detail branch conventions, commit squash rules, and signature verification requirements.</instructions><output_format>CONTRIBUTING.md snippet.</output_format><constraints>Rules must be highly visible and direct.</constraints></output>"),
    ("create database replication strategy checklist", "custom", "<thinking>Replication checklist. Primary-Replica lag checks, failover triggers, write forwarding rules validation.</thinking> <output><role>Database Engineer</role><task>Draft replication guidelines.</task><instructions>Provide verification steps for backup lag, read replica capacity checks, dns record updates.</instructions><output_format>Checklist block.</output_format><constraints>Ensure checks account for write consistency issues on read replicas.</constraints></output>"),
    ("draft security review questionnaire for third-party tools", "custom", "<thinking>Vendor security review. Focus on SOC 2 reports, encryption at rest/transit, authentication methods (SAML/SSO), data retention.</thinking> <output><role>Security Compliance Officer</role><task>Create security questionnaire.</task><instructions>List 8 critical security compliance questions targeting SaaS vendor architectures.</instructions><output_format>Compliance document template.</output_format><constraints>Focus on actionable answers, not generic disclosures.</constraints></output>"),
    ("generate QA test design document structure", "custom", "<thinking>QA Test design framework. Scope, test strategy, test matrix, regression criteria, environments.</thinking> <output><role>QA Architect</role><task>Design test document template.</task><instructions>Draft template for test suites defining execution parameters and critical test paths.</instructions><output_format>Structured outline.</output_format><constraints>Make parameters testable.</constraints></output>"),
    ("write security vulnerability disclosure policy template", "custom", "<thinking>Vulnerability disclosure policy. Safe harbor statement, reporting channel, timeline expectations, prohibition of public disclosure before fix.</thinking> <output><role>SecOps Counsel</role><task>Write security.txt template.</task><instructions>Provide structured policies for white-hat security researchers detailing report handling cycles.</instructions><output_format>Markdown disclosure policy.</output_format><constraints>Ensure safe harbor language is explicit and legally protective.</constraints></output>"),
    ("create technical writing glossary of terms", "custom", "<thinking>Glossary creation. Define standard capitalization rules, deprecated words list, acronym rules.</thinking> <output><role>Doc Lead</role><task>Create glossary skeleton.</task><instructions>Define format: Term, Definition, Capitalization Rules, Allowed Synonyms.</instructions><output_format>Glossary table.</output_format><constraints>Keep rules focused on developer doc consistency.</constraints></output>"),
    ("draft vendor SLA review template", "custom", "<thinking>SLA parameters analysis. Service uptime targets (e.g. 99.9%), support response times, service credits calculations for breaches.</thinking> <output><role>IT Sourcing Lead</role><task>Create SLA review checklist.</task><instructions>Provide steps for verifying credit tiers, reporting periods, and definition of outages.</instructions><output_format>Review framework.</output_format><constraints>Ensure calculation parameters are clear.</constraints></output>"),
    ("generate API breaking change notice layout", "custom", "<thinking>API change notice formatting. Clear warnings, target paths, alternative paths mapping, sunset schedule.</thinking> <output><role>DevRel Writer</role><task>Create announcement layout.</task><instructions>Draft template for notifying integrations about payload schema removals.</instructions><output_format>Markdown template with diff highlighting.</output_format><constraints>Migration guides must compare legacy and new fields side by side.</constraints></output>"),
    ("write git workflow guide for team branching", "custom", "<thinking>Team git guide. Trunk-based development vs GitFlow. Recommend trunk-based for CD pipeline velocity.</thinking> <output><role>Principal Engineer</role><task>Write branching guide.</task><instructions>Detail branch naming (feature/, bugfix/), pull request lifecycles, and merge fast-forward requirements.</instructions><output_format>Markdown workflow guide.</output_format><constraints>Keep rules simple and CD pipeline-friendly.</constraints></output>"),
    ("create technical documentation migration plan template", "custom", "<thinking>Docs migration planning. Focus on asset redirects, path mapping, search index rebuilding, page redirects structure.</thinking> <output><role>Documentation Architect</role><task>Draft migration plan template.</task><instructions>Define tracking table for Source URL, Target URL, Status, Redirect code (e.g. 301), Verification status.</instructions><output_format>Migration sheet table.</output_format><constraints>All redirects must favor 301 permanent redirect rules.</constraints></output>"),
    ("generate product release communication checklist", "custom", "<thinking>Release communications check. Tasks for Product Marketing, Customer Success, sales enablement, documentation deploy sync.</thinking> <output><role>Release Manager</role><task>Draft communications checklist.</task><instructions>Provide list of deliverables sorted by stakeholder group (Marketing, Success, Support, Sales).</instructions><output_format>Markdown roadmap grid.</output_format><constraints>Verify all internal enablement tasks are completed before publishing public notes.</constraints></output>"),
    ("create data dictionary schema structure", "custom", "<thinking>Data dictionary design. Define columns: Table Name, Column Name, Data Type, Constraints, Description, Reference Table/Column.</thinking> <output><role>Database Administrator</role><task>Design data dictionary layout.</task><instructions>Provide a structured table format with examples of primary keys, foreign keys, and descriptions.</instructions><output_format>Data dictionary schema template.</output_format><constraints>Ensure format is clean and documentation-ready.</constraints></output>"),
    ("draft code deprecation strategy document", "custom", "<thinking>Code deprecation strategy. Annotations use, compiler warning targets, deprecation cycle stages, logging deprecation warnings in production.</thinking> <output><role>Principal Architect</role><task>Write deprecation guidelines.</task><instructions>Outline 3-stage deprecation process: 1. Warning annotation, 2. Runtime logging, 3. Actual removal.</instructions><output_format>Engineering standards document.</output_format><constraints>Ensure stages are clearly separated in time.</constraints></output>"),
    ("generate system maintenance window announcement layout", "custom", "<thinking>Maintenance announcement formatting. Specify date, time in UTC, impact (service downtime, read-only mode), status page link.</thinking> <output><role>Operations lead</role><task>Create maintenance banner copy.</task><instructions>Draft email and UI banner templates explaining scheduled database upgrade window.</instructions><output_format>Formatted copy templates.</output_format><constraints>Uptime impacts must be stated in bold text near start.</constraints></output>"),
    ("write developer documentation feedback template", "custom", "<thinking>Documentation feedback collection. Rating scale, helpful/not helpful options, open comments, context logging.</thinking> <output><role>UX Doc Researcher</role><task>Design feedback widget fields.</task><instructions>Draft questions targeting doc clarity, accuracy, and completeness with mock feedback logs.</instructions><output_format>UX form definition.</output_format><constraints>Widget must not interfere with reading flow.</constraints></output>"),
    ("create API rate limit notification template", "custom", "<thinking>API rate limit headers and custom response block when limit is exceeded. Design JSON payload and header specifications.</thinking> <output><role>API Gateway Engineer</role><task>Draft API response structure for HTTP 429.</task><instructions>Define HTTP headers (X-RateLimit-Limit, X-RateLimit-Remaining, Retry-After) and JSON error response formatting.</instructions><output_format>JSON and HTTP specification draft.</output_format><constraints>All retry instructions must be defined in seconds.</constraints></output>"),

    # =========================================================================
    # CODE (36 examples total)
    # =========================================================================
    (
        "parse csv file in python",
        "code",
        "<thinking>Python CSV parsing can use csv.reader or pandas. Since no specific dependency is specified, it is best to provide the built-in csv solution for zero dependencies and pandas for data analysis context.</thinking> <output><role>Senior Python Engineer</role><context>Read transaction dataset in a memory-efficient manner.</context><task>Implement a Python function to parse CSV rows.</task><instructions>1. Use the built-in csv module. 2. Handle potential file encoding issues (utf-8 with BOM). 3. Implement error handling for missing files and malformed rows. 4. Provide a streaming generator function to avoid high memory usage for large files.</instructions><output_format>Python file with type hints and usage example.</output_format><constraints>No external libraries allowed in the main solution.</constraints></output>"
    ),
    (
        "implement a rate limiter in go",
        "code",
        "<thinking>A rate limiter in Go should ideally use the token bucket algorithm. The solution must use standard sync primitives and handle concurrent requests safely.</thinking> <output><role>Principal Go Systems Developer</role><context>Protecting api endpoints from request bursts.</context><task>Write a thread-safe token bucket rate limiter in Go.</task><instructions>1. Avoid sleep loops; calculate token accumulation based on time elapsed since last request. 2. Use sync.Mutex to protect shared state (tokens, lastRefill). 3. Support a dynamic configuration (capacity, refill rate). 4. Provide unit tests with mock time triggers.</instructions><output_format>Complete Go file with package declarations, struct definitions, and main execution verification.</output_format><constraints>Do not use golang.org/x/time/rate; implement the token bucket logic from scratch using basic Go primitives.</constraints></output>"
    ),
    (
        "javascript debounce function",
        "code",
        "<thinking>Debounce limits execution rate of a function until a quiet period has passed. Must handle scoping (this context) and argument passing.</thinking> <output><role>Senior Frontend Engineer</role><context>Optimize browser window resize or input search typing events.</context><task>Implement a debounce function in vanilla JavaScript.</task><instructions>1. Return a wrapper function. 2. Handle timer clearing and resetting. 3. Ensure the original function receives the correct context (this) and arguments. 4. Provide an optional 'immediate' flag to invoke the function at the start of the timeout.</instructions><output_format>Vanilla JavaScript ES6 implementation with use-case examples.</output_format><constraints>Do not use lodash or external dependencies. Must work in modern browser environments.</constraints></output>"
    ),
    (
        "validate phone number in javascript",
        "code",
        "<thinking>Phone number validation requires regular expressions that support international formats (+CountryCode) while avoiding rigid constraints that reject valid inputs.</thinking> <output><role>Full Stack JS Engineer</role><task>Write a JavaScript function to validate E.164 phone formats.</task><instructions>1. Use the E.164 format regex pattern. 2. Provide tests for different countries (+1, +44, +90). 3. Discuss limitations of regex vs libphonenumber library. 4. Return boolean value.</instructions><output_format>Clean JS module with utility functions and test cases.</output_format><constraints>Must not allow arbitrary non-numeric characters beyond spacing and dashes.</constraints></output>"
    ),
    (
        "binary search implementation",
        "code",
        "<thinking>Binary search works on sorted arrays. Explain pre-requisites (sorting), prevent integer overflow during mid calculation, and state complexity.</thinking> <output><role>Algorithms Specialist</role><task>Implement binary search in Python.</task><instructions>1. Handle array boundary conditions (empty lists, duplicates). 2. Use safe middle index calculation: low + (high - low) // 2 to avoid potential overflow. 3. Analyze time and space complexity. 4. Include interactive test assertions.</instructions><output_format>Python script with detailed docstring and tests.</output_format><constraints>Array must be assumed to be already sorted. Do not use built-in bisect library.</constraints></output>"
    ),
    (
        "connect to redis in python",
        "code",
        "<thinking>Redis connection in Python uses redis-py. Best practice is connection pooling, handling ConnectionError, and using context managers.</thinking> <output><role>Backend Engineer</role><task>Write a Redis connection pool class in Python.</task><instructions>1. Use redis-py library. 2. Implement connection pooling. 3. Include error handling for host unreachable/timeouts. 4. Implement a simple key set/get wrapper with serialization.</instructions><output_format>Modular python class with connection lifecycle management.</output_format><constraints>Avoid creating new connections on every function call; reuse the pool.</constraints></output>"
    ),
    # Additional code
    ("parse XML file in Python", "code", "<thinking>Python XML parsing can use xml.etree.ElementTree. Ensure error handling for malformed tags.</thinking> <output><role>Senior Engineer</role><task>Parse XML in Python.</task><instructions>Use standard library ElementTree. Implement try-except for XML syntax validation.</instructions><output_format>Python function.</output_format><constraints>Zero external packages.</constraints></output>"),
    ("implement a thread pool in Go", "code", "<thinking>Go concurrency primitives. Use worker channels, sync.WaitGroup for tracking jobs completion.</thinking> <output><role>Go Systems Developer</role><task>Write a concurrent job pool.</task><instructions>Implement worker functions reading from a shared jobs channel. Use context cancellation.</instructions><output_format>Executable Go file.</output_format><constraints>Ensure safe clean exit of worker routines.</constraints></output>"),
    ("javascript throttle function", "code", "<thinking>Throttle execution limits to one execution per time interval. Handle trailing edge arguments.</thinking> <output><role>JS Dev</role><task>Implement throttle.</task><instructions>Write throttle wrapper handling timer flags and context scoping.</instructions><output_format>ES6 Javascript.</output_format><constraints>No lodash.</constraints></output>"),
    ("validate email address in Javascript", "code", "<thinking>Standard regex validation for email strings. Cover edge cases of domains.</thinking> <output><role>Frontend Developer</role><task>Validate email.</task><instructions>Write regex-based evaluation function.</instructions><output_format>JS utility function.</output_format><constraints>Do not reject valid plus-sign email extensions.</constraints></output>"),
    ("quick sort algorithm implementation", "code", "<thinking>Quick sort requires partitioning element select, recursive array divisions.</thinking> <output><role>Algorithm Dev</role><task>Implement Quick Sort.</task><instructions>Write quicksort in Python using in-place partitioning with pivot logic.</instructions><output_format>Python script.</output_format><constraints>Analyze time complexity in docstring.</constraints></output>"),
    ("connect to PostgreSQL database in Python", "code", "<thinking>Python Postgres. Use psycopg2 or asyncpg. Establish connection parameters safely.</thinking> <output><role>Database Python Dev</role><task>Connect to Postgres.</task><instructions>Use psycopg2 driver with connection pooling classes.</instructions><output_format>Python class.</output_format><constraints>Never hardcode host credentials; read from env.</constraints></output>"),
    ("implement stack data structure in Python", "code", "<thinking>Stack operations. Push, Pop, Peek, isEmpty. Implement using list array representation.</thinking> <output><role>Developer</role><task>Implement Stack.</task><instructions>Write a Python class implementing stack methods with list append/pop operations.</instructions><output_format>Class definition.</output_format><constraints>Handle popping from empty stack error.</constraints></output>"),
    ("write a REST handler in Go", "code", "<thinking>Go HTTP programming. Use net/http package. Handle routing parameters and JSON serialization.</thinking> <output><role>Go Backend Engineer</role><task>Write HTTP Handler.</task><instructions>Implement http.HandlerFunc reading user body, validating fields, and writing JSON response.</instructions><output_format>Go package code.</output_format><constraints>Only use built-in packages.</constraints></output>"),
    ("javascript sleep function promise", "code", "<thinking>Javascript asynchronous sleep. Return Promise resolved after setTimeout duration.</thinking> <output><role>JS Developer</role><task>Promise-based sleep.</task><instructions>Write ES6 helper function returning a Promise using setTimeout.</instructions><output_format>Helper function block.</output_format><constraints>Ensure function can be awaited.</constraints></output>"),
    ("read file line by line in Python", "code", "<thinking>File operations. Memory-efficient line streaming in Python using open generator.</thinking> <output><role>Python Engineer</role><task>Read file lines.</task><instructions>Write context manager file open with generator yield loop.</instructions><output_format>Python code.</output_format><constraints>Must handle files too large for RAM.</constraints></output>"),
    ("merge two sorted arrays in JavaScript", "code", "<thinking>Sorted array merging. Linear traversal with two pointers. Time complexity O(n + m).</thinking> <output><role>Frontend Lead</role><task>Merge sorted arrays.</task><instructions>Write JavaScript algorithm combining arrays maintaining sorted constraint.</instructions><output_format>JS utility.</output_format><constraints>Do not use built-in array sort after concatenation.</constraints></output>"),
    ("implement binary search tree in Python", "code", "<thinking>BST structure. Node class with left/right branches, insert, and search implementations.</thinking> <output><role>Python dev</role><task>Implement BST.</task><instructions>Write Node class and BST class with recursive search/insert operations.</instructions><output_format>Python file.</output_format><constraints>Document structure clearly.</constraints></output>"),
    ("write a basic websocket server in Nodejs", "code", "<thinking>Node.js Websockets. Use ws library. Handle connection open, message, close events.</thinking> <output><role>Node.js Dev</role><task>Write Websocket Server.</task><instructions>Implement ws module server listening on port, echo back received messages.</instructions><output_format>Node.js script.</output_format><constraints>Provide standard npm install instruction in comments.</constraints></output>"),
    ("find duplicates in list Python", "code", "<thinking>Duplicate search. Using set traversal to achieve O(n) runtime complexity.</thinking> <output><role>Developer</role><task>Find duplicates.</task><instructions>Write helper function returning set of duplicate elements in an input list.</instructions><output_format>Python utility.</output_format><constraints>Must run in linear time.</constraints></output>"),
    ("send HTTP post request in Go", "code", "<thinking>Go HTTP client. Send POST payload, handle response bytes and connection close defer.</thinking> <output><role>Go Lead</role><task>HTTP POST Client.</task><instructions>Use net/http client to send JSON payload. Read response body safely.</instructions><output_format>Go code.</output_format><constraints>Decompress responses if server encoding dictates.</constraints></output>"),
    ("reverse a linked list in Python", "code", "<thinking>Linked list traversal. Iterative node updates (previous, current, next) pointer operations.</thinking> <output><role>Engineer</role><task>Reverse linked list.</task><instructions>Write iterative Python function swapping list direction.</instructions><output_format>Python utility.</output_format><constraints>Space complexity must be O(1).</constraints></output>"),
    ("javascript deep clone object", "code", "<thinking>Deep cloning objects. Handle nested structures, arrays, dates, maps without JSON.stringify limitations.</thinking> <output><role>JS dev</role><task>Deep clone object.</task><instructions>Write custom recursive clone utility checking item prototypes.</instructions><output_format>JS file.</output_format><constraints>Do not lose functional references in child properties.</constraints></output>"),
    ("parse URL query parameters in Go", "code", "<thinking>Go URL parsing. Use net/url package ParseQuery methods.</thinking> <output><role>Go engineer</role><task>Parse query string.</task><instructions>Write Go handler decoding query parameters into custom map data structures.</instructions><output_format>Go code.</output_format><constraints>Verify proper key presence check logic.</constraints></output>"),
    ("read JSON file python", "code", "<thinking>Python JSON read. Use json.load, context manager open syntax.</thinking> <output><role>Developer</role><task>Read JSON.</task><instructions>Write function loading dictionary configurations from file path.</instructions><output_format>Python function.</output_format><constraints>Add JSONDecodeError validation try-except block.</constraints></output>"),
    ("implement bubble sort algorithm Python", "code", "<thinking>Bubble sort. Nested array scans swapping adjacent elements, optimization flag for sorted sweeps.</thinking> <output><role>Teacher</role><task>Implement Bubble Sort.</task><instructions>Write classic bubble sort in Python with a boolean sorted flag check.</instructions><output_format>Python code.</output_format><constraints>Time complexity must be explained.</constraints></output>"),
    ("convert string to integer javascript", "code", "<thinking>Javascript string parsing. Use parseInt with explicit radix (base 10), checking NaN cases.</thinking> <output><role>Frontend dev</role><task>Convert string to integer.</task><instructions>Write function converting string, returning custom default if NaN.</instructions><output_format>JS function.</output_format><constraints>Radix parameter must be defined as 10.</constraints></output>"),
    ("implement queue data structure Go", "code", "<thinking>Go queue representation. Use slice buffer with Mutex synchronization protection.</thinking> <output><role>Go Dev</role><task>Implement Thread-safe Queue.</task><instructions>Write Queue struct with Enqueue and Dequeue methods protected by Mutex locks.</instructions><output_format>Go package code.</output_format><constraints>Support generic structures if possible.</constraints></output>"),
    ("write a simple TCP server in Python", "code", "<thinking>Python socket API. Listen, bind, accept client connections, thread spawns for multi-client handles.</thinking> <output><role>Network python Dev</role><task>Write TCP Server.</task><instructions>Use socket library. Listen on local port, accept loop dispatching threads.</instructions><output_format>Python script.</output_format><constraints>Close sockets on shutdown hook intercept.</constraints></output>"),
    ("flatten nested array javascript", "code", "<thinking>JS array flattening. Use recursive reduce/concat or Array.prototype.flat.</thinking> <output><role>JS Dev</role><task>Flatten array.</task><instructions>Write custom recursive array flatten function supporting depth parameter.</instructions><output_format>JS function.</output_format><constraints>Do not rely on flat() native call directly.</constraints></output>"),
    ("hash a password using bcrypt in Go", "code", "<thinking>Go cryptography. Use golang.org/x/crypto/bcrypt hash generation/comparison methods.</thinking> <output><role>SecOps developer</role><task>Hash passwords in Go.</task><instructions>Write password hash helper returning bytes and login match checking boolean.</instructions><output_format>Go code.</output_format><constraints>Check execution cost values.</constraints></output>"),
    ("implement DFS graph traversal Python", "code", "<thinking>DFS graph search. Use recursive traversal path checks or iterative tracking using manual stack.</thinking> <output><role>Python Engineer</role><task>DFS graph traversal.</task><instructions>Write traversal function logging visited nodes using adjacency lists.</instructions><output_format>Python function.</output_format><constraints>Represent graph as adjacency map dictionary.</constraints></output>"),
    ("check if string is palindrome JavaScript", "code", "<thinking>Palindrome validation. Remove non-alphanumeric chars, case normalization, matching string reverse.</thinking> <output><role>Frontend developer</role><task>Verify Palindrome.</task><instructions>Write JS function cleaning non-alphanumeric text, checking characters match.</instructions><output_format>JS utility.</output_format><constraints>Whitespace must be ignored.</constraints></output>"),
    ("parse CLI flags in Go", "code", "<thinking>Go CLI flags. Use built-in flag package. Set pointer binds, call parse.</thinking> <output><role>Go Developer</role><task>Parse Go CLI flags.</task><instructions>Write Go function defining string/int flags, parsing and displaying results.</instructions><output_format>Go main file.</output_format><constraints>Provide default values.</constraints></output>"),
    ("write a custom middleware in Express Nodejs", "code", "<thinking>Express JS middleware structure. Request, response, next callback logic. Access logs example.</thinking> <output><role>Node.js engineer</role><task>Create Express Middleware.</task><instructions>Write logging middleware tracking HTTP method, URL path, response execution duration.</instructions><output_format>Express middleware function.</output_format><constraints>Must call next() to prevent request hangs.</constraints></output>"),
    ("generate random string Python", "code", "<thinking>Python secure random strings. Use secrets module for cryptographically secure values.</thinking> <output><role>SecOps Python Dev</role><task>Generate secure token.</task><instructions>Write function using secrets.token_hex returning secure hex strings.</instructions><output_format>Python script.</output_format><constraints>Do not use standard random module due to seed predictability.</constraints></output>"),

    # =========================================================================
    # ACADEMIC (36 examples total)
    # =========================================================================
    (
        "explain cognitive dissonance theory",
        "academic",
        "<thinking>This is a fundamental social psychology concept. I must prompt for formal theory definition, key researchers (Leon Festinger), mechanisms of reduction, and peer-reviewed studies.</thinking> <output><role>Social Psychology Professor</role><task>Write a scholarly overview of Cognitive Dissonance Theory.</task><instructions>1. Introduce Leon Festinger's core hypothesis. 2. Define the cognitive mechanism (inconsistency between beliefs and actions causing discomfort). 3. Detail the 3 primary methods individuals use to reduce dissonance. 4. Cite the classic 1959 experiment ('Cognitive Consequences of Forced Compliance') and summarize its findings. 5. Discuss modern developments in the theory.</instructions><output_format>Structured academic overview with APA format references.</output_format><constraints>Maintain an objective, academic tone. Cite researchers accurately.</constraints></output>"
    ),
    (
        "how to write a literature gap statement",
        "academic",
        "<thinking>Literature gaps form the core of a research paper's significance. Guide the model to teach structured rhetorical moves (CARS model - Creating a Research Space) and transition terms.</thinking> <output><role>Academic Writing Instructor</role><task>Formulate guide for writing a research gap statement.</task><instructions>1. Explain John Swales' CARS model (specifically Move 2: Establishing a Niche). 2. Identify common types of gaps (empirical, methodological, theoretical). 3. Provide a list of academic transition phrases to signal gaps (e.g., 'However, research has yet to consider...'). 4. Show a before/after example of a weak vs strong gap statement.</instructions><output_format>Technical writing guide with analytical examples.</output_format><constraints>Focus strictly on scholarly writing standards, not blog post intros.</constraints></output>"
    ),
    (
        "define qualitative research triangulation",
        "academic",
        "<thinking>Triangulation in qualitative study ensures credibility and validity. Explain types: data, investigator, theory, and methodological triangulation.</thinking> <output><role>Research Methodology Specialist</role><task>Explain the concept of triangulation in qualitative research.</task><instructions>1. Define triangulation and its purpose in qualitative design. 2. Detail the four types of triangulation based on Denzin's classification. 3. Discuss how triangulation improves validity and minimizes researcher bias. 4. Address limitations (e.g., increased resource costs, conflicting data resolution).</instructions><output_format>Methodological reference note.</output_format><constraints>Maintain strict methodological terminology. Cite academic research frameworks.</constraints></output>"
    ),
    (
        "explain double-blind study design",
        "academic",
        "<thinking>Explain clinical study methodology, bias prevention (placebo effect, researcher bias, observer bias), randomization, and ethical/operational limits.</thinking> <output><role>Clinical Trials Methodologist</role><task>Explain the double-blind randomized controlled trial (RCT) design.</task><instructions>1. Define double-blind in contrast to single-blind and open-label designs. 2. Explain how this design mitigates both participant placebo effects and observer bias. 3. Detail the randomization process and the role of independent third-party monitoring. 4. Discuss ethical boundaries where double-blinding is unfeasible.</instructions><output_format>Research design handbook extract.</output_format><constraints>Use standard epidemiological terms (randomization, double-dummy, allocation concealment).</constraints></output>"
    ),
    (
        "what is academic peer review process",
        "academic",
        "<thinking>Explain the workflow of peer review: editor triage, reviewer invitation, blind styles (single, double, open), feedback categories, and decision cycles.</thinking> <output><role>Journal Editor-in-Chief</role><task>Detail the scholarly peer review workflow.</task><instructions>1. Walk through chronological stages: manuscript submission, initial desk review/triage, peer assignment, evaluation, revision cycles, and final decision. 2. Compare Single-Blind, Double-Blind, and Open Peer Review models. 3. Discuss issues facing the current peer review model (reviewer fatigue, bias) and proposed reforms.</instructions><output_format>Publishing guide block.</output_format><constraints>Maintain an institutional, publishing-industry perspective. Do not include informal blogs.</constraints></output>"
    ),
    (
        "explain keynesian multiplier effect",
        "academic",
        "<thinking>Economic theory. Marginal propensity to consume (MPC), spending injection cycle, mathematical calculation formula, and policy criticisms.</thinking> <output><role>Professor of Macroeconomics</role><task>Explain the Keynesian multiplier effect mathematically and theoretically.</task><instructions>1. Define the multiplier effect and state the mathematical formula: Mult = 1 / (1 - MPC). 2. Explain the marginal propensity to consume (MPC) and marginal propensity to save (MPS). 3. Walk through a step-by-step example of how a government spending injection propagates through the economy. 4. Discuss the crowding-out effect and other neoclassical criticisms of the multiplier.</instructions><output_format>Economic theory lecture summary with formula derivation.</output_format><constraints>Formula derivations must be algebraically correct and explained.</constraints></output>"
    ),
    # Additional academic
    ("explain social identity theory", "academic", "<thinking>Social psychology. Henri Tajfel, categorization, identification, comparison mechanisms, in-group favoritism.</thinking> <output><role>Psychology Professor</role><task>Explain Social Identity Theory.</task><instructions>1. Introduce Tajfel and Turner origins. 2. Define the three cognitive phases of identity construction. 3. Explain bias outcomes.</instructions><output_format>Theoretical overview.</output_format><constraints>Must remain academic, cite standard experiments.</constraints></output>"),
    ("how to write a research methodology outline", "academic", "<thinking>Research design structuring. Outline participants, tools, procedure, database parameters.</thinking> <output><role>Methodology Instructor</role><task>Write methodology outline template.</task><instructions>Draft structured list detailing sampling strategies and regression setups.</instructions><output_format>Thesis outline template.</output_format><constraints>Exclude actual results data reporting.</constraints></output>"),
    ("define qualitative content analysis coding", "academic", "<thinking>Qualitative text coding. Deductive vs inductive codes, inter-coder reliability, theme extraction.</thinking> <output><role>Social Scientist</role><task>Explain content coding.</task><instructions>1. Differentiate coding styles. 2. Detail codebook definition. 3. Discuss reliability checks.</instructions><output_format>Academic brief.</output_format><constraints>Cite standard qualitative textbooks (Saldaña).</constraints></output>"),
    ("explain quasi-experimental study design", "academic", "<thinking>Research design. Lack of random assignment, comparison groups, threat to internal validity.</thinking> <output><role>Methodologist</role><task>Explain quasi-experiments.</task><instructions>1. Differentiate from true experiments. 2. Explain assignment methods. 3. Detail threats like selection bias.</instructions><output_format>Methodology notes.</output_format><constraints>Ensure accurate statistical definitions.</constraints></output>"),
    ("what is the scientific replication crisis", "academic", "<thinking>Scientific methodology crisis. Low power studies, publication bias, p-hacking, reproduction attempts failure.</thinking> <output><role>Meta-Science Researcher</role><task>Explain replication crisis.</task><instructions>1. Define causes (p-hacking, publishing bias). 2. Cite open science framework reforms. 3. Explain preregistration role.</instructions><output_format>Critical review.</output_format><constraints>Maintain balanced perspective across fields (psychology vs oncology).</constraints></output>"),
    ("explain modern monetary theory basics", "academic", "<thinking>Economics. Sovereign currency issuers, deficit spending limits, inflation control via taxation.</thinking> <output><role>Economics Instructor</role><task>Explain MMT framework.</task><instructions>1. Define monetary sovereignty. 2. State MMT view of tax utility. 3. Address neoclassical inflation arguments.</instructions><output_format>Theoretical summary.</output_format><constraints>Describe both MMT proposals and orthodox criticisms neutrally.</constraints></output>"),
    ("define construct validity in social sciences", "academic", "<thinking>Measurement theory. How well a scale measures the target abstract construct. Divergent vs convergent validity.</thinking> <output><role>Psychometrician</role><task>Explain Construct Validity.</task><instructions>1. Define the concept in measurement. 2. Compare convergent and discriminant validation methods. 3. Discuss factor analysis role.</instructions><output_format>Academic review.</output_format><constraints>Cite Cronbach and Meehl foundational concepts.</constraints></output>"),
    ("explain structure agency debate sociology", "academic", "<thinking>Sociological theory. Individual autonomy vs structural social formatting constraints (Giddens, Bourdieu).</thinking> <output><role>Sociology Professor</role><task>Explain Structure vs Agency.</task><instructions>1. Outline historical debates. 2. Detail Giddens' structuration theory. 3. Discuss Bourdieu's concept of habitus.</instructions><output_format>Theoretical briefing.</output_format><constraints>Keep sociological terminology standard.</constraints></output>"),
    ("how to formulate a research hypothesis", "academic", "<thinking>Scientific research formulation. Testable, directional, falsifiable statements based on theory.</thinking> <output><role>Academic Writing Coach</role><task>Guide on hypothesis formulation.</task><instructions>1. Define null vs alternative hypotheses. 2. List criteria for hypothesis validity. 3. Provide examples of weak vs strong formulations.</instructions><output_format>Structured guide.</output_format><constraints>Focus on empirical validation design requirements.</constraints></output>"),
    ("define standard error of measurement", "academic", "<thinking>Psychometrics. Standard deviation of errors in measurement score distributions. Formula connecting SD and reliability.</thinking> <output><role>Psychometric Analyst</role><task>Explain SEM measurement error.</task><instructions>1. State mathematical formula. 2. Explain how it establishes score confidence intervals. 3. Contrast with standard error of estimate.</instructions><output_format>Statistical note.</output_format><constraints>Provide formula notation definition.</constraints></output>"),
    ("explain historical materialism theory", "academic", "<thinking>Political economy theory. Marxist framework where economic mode of production shapes social structure.</thinking> <output><role>Political Science Professor</role><task>Explain Historical Materialism.</task><instructions>1. Define base and superstructure. 2. Describe historical progression through class conflicts. 3. Contrast with Hegelian idealism.</instructions><output_format>Theoretical primer.</output_format><constraints>Maintain academic historical vocabulary.</constraints></output>"),
    ("define semantic memory in psychology", "academic", "<thinking>Cognitive psychology. Long-term memory for facts, concepts, language, separate from episodic memory.</thinking> <output><role>Cognitive Neuroscientist</role><task>Explain Semantic Memory.</task><instructions>1. Contrast with episodic memory. 2. Describe storage structures (spreading activation model). 3. Identify brain regions involved (temporal lobe).</instructions><output_format>Cognitive review.</output_format><constraints>Cite Tulving's categorization frameworks.</constraints></output>"),
    ("explain the role of the peer reviewer", "academic", "<thinking>Academic publishing. Evaluation checklist, constructive criticisms, recommendations rationale.</thinking> <output><role>Journal Editor</role><task>Explain reviewer obligations.</task><instructions>1. Outline evaluation criteria. 2. Detail how to identify duplicate publications. 3. Discuss guidelines for conflict of interest declarations.</instructions><output_format>Reviewer guide.</output_format><constraints>Maintain professional standard guidelines.</constraints></output>"),
    ("what is the hawthorne effect", "academic", "<thinking>Research methodology bias. Behavioral changes in study participants due to the awareness of being observed.</thinking> <output><role>Organizational Psychologist</role><task>Explain Hawthorne Effect.</task><instructions>1. Detail the historical factory experiment. 2. Analyze the psychological cause. 3. Outline design mitigations in research.</instructions><output_format>Methodological overview.</output_format><constraints>Avoid simplifying as just 'worker productivity increase'.</constraints></output>"),
    ("explain critical race theory origin", "academic", "<thinking>Legal and sociological theory. Examination of systemic racism within legal structures, Derrick Bell, Kimberlé Crenshaw.</thinking> <output><role>Legal Studies Professor</role><task>Detail CRT origin and tenets.</task><instructions>1. Cite 1970s legal scholarship beginnings. 2. Explain concepts of intersectionality and interest convergence. 3. Explain how it evaluates legal neutrality.</instructions><output_format>Scholarly overview.</output_format><constraints>Maintain strict legal and sociological vocabulary.</constraints></output>"),
    ("define internal consistency reliability Cronbach alpha", "academic", "<thinking>Statistics. Measurement of scale items inter-correlations, calculation limitations, alpha values.</thinking> <output><role>Statistician</role><task>Explain Cronbach's Alpha.</task><instructions>1. Write formula. 2. Explain alpha value targets. 3. Discuss pitfalls (e.g. scale length inflation).</instructions><output_format>Mathematical note.</output_format><constraints>Highlight that alpha is not a measure of unidimensionality.</constraints></output>"),
    ("explain panopticon concept in philosophy", "academic", "<thinking>Philosophical surveillance theory. Bentham architecture, Michel Foucault's analysis of power internalization.</thinking> <output><role>Philosophy Professor</role><task>Explain Panopticism.</task><instructions>1. Describe the structural design of Bentham's prison. 2. Detail Foucault's extension to modern social institutions. 3. Discuss self-policing outcomes.</instructions><output_format>Philosophical essay.</output_format><constraints>Use standard Foucault terminology (discipline, gaze).</constraints></output>"),
    ("how to outline a thesis introduction", "academic", "<thinking>Thesis structure. Funnel style: general context, literature niche, problem, significance roadmap.</thinking> <output><role>Academic Writing Advisor</role><task>Draft thesis introduction template.</task><instructions>Outline the CARS (Creating a Research Space) framework for the introductory chapter.</instructions><output_format>Structured outline.</output_format><constraints>Include specific chapter roadmap guidance.</constraints></output>"),
    ("define ethnography qualitative method", "academic", "<thinking>Qualitative research. Immersive participant observation, field notes, reflexivity considerations.</thinking> <output><role>Anthropologist</role><task>Explain Ethnographic Methods.</task><instructions>1. Define participant observation. 2. Explain the necessity of keeping field notes. 3. Discuss researcher reflexivity and ethics.</instructions><output_format>Methodological brief.</output_format><constraints>Keep focus on anthropological rigor.</constraints></output>"),
    ("explain schema theory cognitive psychology", "academic", "<thinking>Cognitive psychology. Mental structures organizing information, assimilation vs accommodation (Piaget).</thinking> <output><role>Cognitive Psychologist</role><task>Explain Schema Theory.</task><instructions>1. Define schema. 2. Contrast Piaget's concepts of assimilation and accommodation. 3. Discuss impact on memory recall distortions.</instructions><output_format>Theoretical outline.</output_format><constraints>Include memory retrieval impact.</constraints></output>"),
    ("what is statistical power calculation", "academic", "<thinking>Statistics. Probability of rejecting null hypothesis when it is false (1 - beta), dependent variables.</thinking> <output><role>Quantitative Analyst</role><task>Explain Statistical Power.</task><instructions>1. Define type II error (beta). 2. Explain the four variables of power analysis: alpha, sample size, effect size, power. 3. Explain how to determine required sample size.</instructions><output_format>Methodological note.</output_format><constraints>Formulas must align with G*Power standard inputs.</constraints></output>"),
    ("explain mercantilism economic theory", "academic", "<thinking>Economic history. Wealth accumulation (gold/silver), trade surplus targets, protectionism, colonial monopolies.</thinking> <output><role>Economic Historian</role><task>Explain Mercantilism.</task><instructions>1. Detail core doctrines. 2. Explain historical period (16th-18th centuries). 3. Contrast with Adam Smith's free market theories.</instructions><output_format>Historical summary.</output_format><constraints>Use correct historical terms.</constraints></output>"),
    ("define discourse analysis methodology", "academic", "<thinking>Qualitative research. Study of language in social contexts, power dynamics (Foucault, Fairclough).</thinking> <output><role>Linguistic Researcher</role><task>Explain Discourse Analysis.</task><instructions>1. Define textual vs social discourse. 2. Explain critical discourse analysis principles. 3. Outline steps for analysis.</instructions><output_format>Methodology primer.</output_format><constraints>Distinguish from plain content analysis.</constraints></output>"),
    ("explain Maslow hierarchy of needs criticisms", "academic", "<thinking>Psychology theory review. Criticisms of Maslow's model: lack of empirical validation, cultural bias, rigidity of levels.</thinking> <output><role>Clinical Psychologist</role><task>Critique Maslow's Hierarchy.</task><instructions>1. Briefly present original model. 2. Detail structural critiques (rigid hierarchy). 3. Discuss individual variation examples from collectivistic societies.</instructions><output_format>Theoretical critique.</output_format><constraints>Maintain respectful scholarly critique.</constraints></output>"),
    ("define convenience sampling bias", "academic", "<thinking>Sampling methodology. Non-probability selection based on accessibility, threats to external validity.</thinking> <output><role>Methodology Consultant</role><task>Explain Convenience Sampling bias.</task><instructions>1. Define sampling method. 2. Detail how it harms external validity (generalizability). 3. Suggest probability sampling alternatives.</instructions><output_format>Methodology warning card.</output_format><constraints>Explain differences in statistical inference limits.</constraints></output>"),
    ("explain dependency theory international relations", "academic", "<thinking>International political economy. Flow of resources from periphery poor states to core wealthy states, structural inequality.</thinking> <output><role>IR Professor</role><task>Explain Dependency Theory.</task><instructions>1. Identify the core-periphery model. 2. Detail how international trade terms favor wealthy economies. 3. Contrast with modernization theory.</instructions><output_format>IR theory notes.</output_format><constraints>Keep focus on political-economic structures.</constraints></output>"),
    ("how to draft an academic poster abstract", "academic", "<thinking>Academic abstracts for conference posters. Need short problem, method, findings, and conclusion.</thinking> <output><role>Conference Prep Coach</role><task>Guide for poster abstract.</task><instructions>Outline the 5 key sentences: Context, Gap, Method, Result, Implication under a 150-word cap.</instructions><output_format>Writing brief template.</output_format><constraints>Abstract structure must fit typical academic submission limits.</constraints></output>"),
    ("define institutionalism in political science", "academic", "<thinking>Political theory. Study of formal and informal rules and organizations, new institutionalism types.</thinking> <output><role>Political Scientist</role><task>Explain Institutionalism.</task><instructions>1. Define institution in political analysis. 2. Compare historical, rational choice, and sociological institutionalism.</instructions><output_format>Theoretical briefing.</output_format><constraints>Use proper classification terms.</constraints></output>"),
    ("explain self-determination theory motivation", "academic", "<thinking>Psychology of motivation. Deci and Ryan, intrinsic vs extrinsic, 3 basic needs (autonomy, competence, relatedness).</thinking> <output><role>Motivation Researcher</role><task>Explain Self-Determination Theory.</task><instructions>1. List the three basic psychological needs. 2. Describe the autonomy-to-control continuum of motivation.</instructions><output_format>Theoretical primer.</output_format><constraints>Autonomy must be clearly distinguished from independence.</constraints></output>"),
    ("what is the difference between inductive and deductive reasoning", "academic", "<thinking>Epistemology/Philosophy of science. Bottom-up theory building from observation vs top-down theory testing from general rules.</thinking> <output><role>Epistemology Professor</role><task>Contrast Inductive and Deductive reasoning.</task><instructions>1. Define both reasoning structures. 2. Contrast reliability of conclusions (probability vs absolute certainty). 3. Provide examples of scientific cycle loops integrating both.</instructions><output_format>Scholarly primer.</output_format><constraints>Make logic steps explicit.</constraints></output>")
]


def clean_and_normalize_existing():
    """Reads existing CSV, normalizes template labels, removes duplicates."""
    if not os.path.exists(CSV_PATH):
        print(f"[!] Error: CSV file not found at {CSV_PATH}")
        return []

    unique_rows = {}
    print(f"[*] Reading existing dataset from: {CSV_PATH}")

    with open(CSV_PATH, "r", encoding="utf-8") as f:
        reader = csv.reader(f)
        try:
            header = next(reader)
        except StopIteration:
            print("[!] Empty CSV file.")
            return []

        # Keep track of header columns
        if header != ["original_prompt", "template", "optimized_prompt"]:
            print(f"[!] Warning: Header is unexpected: {header}")

        for i, row in enumerate(reader, start=2):
            if len(row) < 3:
                continue
            original_prompt = row[0].strip()
            template = row[1].strip().replace('"', '').replace("'", "")
            optimized_prompt = row[2].strip()

            # Deduplicate based on original_prompt (case insensitive key, keeping first)
            key = original_prompt.lower()
            if key not in unique_rows:
                unique_rows[key] = {
                    "original_prompt": original_prompt,
                    "template": template,
                    "optimized_prompt": optimized_prompt
                }

    print(f"[+] Loaded {len(unique_rows)} unique rows from existing dataset.")
    return list(unique_rows.values())


def balance_dataset(existing_rows):
    """Appends synthetic examples to balance categories at 60 rows each."""
    # Count existing rows per template and collect existing prompts
    counts = {}
    existing_prompts = set()
    for r in existing_rows:
        tpl = r["template"]
        counts[tpl] = counts.get(tpl, 0) + 1
        existing_prompts.add(r["original_prompt"].strip().lower())

    print("\n[*] Current template counts (pre-balancing):")
    for tpl, count in sorted(counts.items()):
        print(f"  - {tpl}: {count}")

    # Check which synthetic examples to add
    added_count = 0
    added_by_template = {}

    for orig, tpl, opt in SYNTHETIC_EXAMPLES:
        if orig.strip().lower() in existing_prompts:
            continue
        current_count = counts.get(tpl, 0)
        if current_count < 60:
            existing_rows.append({
                "original_prompt": orig,
                "template": tpl,
                "optimized_prompt": opt
            })
            counts[tpl] = current_count + 1
            existing_prompts.add(orig.strip().lower())
            added_count += 1
            added_by_template[tpl] = added_by_template.get(tpl, 0) + 1

    print(f"\n[+] Added {added_count} synthetic examples:")
    for tpl, count in sorted(added_by_template.items()):
        print(f"  - {tpl}: {count} added")

    print("\n[*] Final template counts (post-balancing):")
    for tpl, count in sorted(counts.items()):
        print(f"  - {tpl}: {count}")

    return existing_rows


def write_dataset(rows):
    """Writes the balanced rows back to the CSV file."""
    print(f"\n[*] Writing {len(rows)} rows back to: {CSV_PATH}")
    with open(CSV_PATH, "w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f, quoting=csv.QUOTE_MINIMAL)
        # Write header
        writer.writerow(["original_prompt", "template", "optimized_prompt"])
        for row in rows:
            writer.writerow([
                row["original_prompt"],
                row["template"],
                row["optimized_prompt"]
            ])
    print("[+] Successfully updated dataset.")


def main():
    existing_rows = clean_and_normalize_existing()
    if not existing_rows:
        print("[!] No rows to process. Aborting.")
        return

    balanced_rows = balance_dataset(existing_rows)
    write_dataset(balanced_rows)


if __name__ == "__main__":
    main()
