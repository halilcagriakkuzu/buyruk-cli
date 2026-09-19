# buyruk-cli

**buyruk** (Old Turkic: *buyruk* [command/decree]) is a local-first, filesystem-based project management and orchestration tool built in Go. It provides a high-performance, Jira-like workflow for developers who prefer the speed of the terminal and the privacy of local storage.

## 1. Vision & Goals

* **Speed:** Near-zero latency through local filesystem operations.
* **Sovereignty:** 100% offline; data stays in human-readable JSON on your machine.
* **Orchestration:** Advanced dependency mapping to manage complex task flows.
* **AI-Native:** Built-in support for token-optimized formats (L-SON) for LLM interactions (e.g., Cursor, Claude, GPT-4).

## 2. Technical Stack

* **Language:** Go (Golang) for single-binary cross-platform distribution.
* **Persistence:** Filesystem-based JSON with a metadata index for $O(1)$ list performance.
* **Target OS:** Windows, macOS, Linux.
* **CLI Framework:** `Cobra`.
* **UI/Rendering:** `Lipgloss` (Colors/Styles), `Tablewriter` (Tables), `Glamour` (Markdown).
* **Repository:** `github.com/halilcagriakkuzu/buyruk-cli`

## 3. Storage & Configuration Architecture

### 3.1 OS-Native Paths

`buyruk` respects system conventions using Go's `os.UserConfigDir()`:

* **Windows:** `%AppData%\buyruk\`
* **Linux/macOS:** `~/.config/buyruk/`

### 3.2 Concurrency & Atomic Writes

To prevent data corruption during simultaneous terminal commands:

1. **Process Locking:** Every write creates a `.buyruk.lock`. If a lock exists, subsequent commands wait/retry for 5 seconds before timeout.
2. **Transaction Log:** A `.buyruk_pending` file records the intent before modification.
3. **Atomic Rename:** Updates are written to `.tmp` files and then renamed (`os.Rename`) to ensure the file is never in a partial state.
4. **Integrity Check:** On startup, if `.buyruk_pending` exists, the tool flags a potential crash and offers a `repair` command.

### 3.3 Directory Structure

```text
[ConfigDir]/buyruk/
├── config.json              # Global defaults (format, project, etc.)
└── projects/
    └── PROJ_KEY/            
        ├── .buyruk.lock     # Concurrency lock
        ├── .buyruk_pending  # Transaction log
        ├── project.json     # INDEX: Registry of all issues (Title, Status, Epic, ID)
        ├── epics/           
        │   └── E-1.json     
        └── issues/          
            ├── T-41.json    # Full Task data (Description, PRs, Deps)
            └── B-12.json    
```

## 4. Functional Requirements

### 4.1 Data Model

* **Attributes:** Title (Required), Status (TODO/DOING/DONE), Priority (LOW to CRITICAL).
* **Metadata:** Markdown Description, PR Link Array, Dependency IDs (`blocked_by`), Epic Link.
* **ID System:** Project-prefixed (e.g., `CORE-12`).

### 4.2 Configuration

Manage defaults via `buyruk config`:

* `buyruk config set default_project <KEY>`
* `buyruk config set default_format <modern|json|lson>`

### 4.3 Command Patterns

All read/listing commands support the `--format` flag to override defaults.

| Command | Action | Format Support | 
| :--- | :--- | :--- | 
| `buyruk list` | List project issues (using index) | Yes | 
| `buyruk view <id>` | Detailed view (using issue file) | Yes | 
| `buyruk task create` | Create a new task | N/A | 
| `buyruk task link` | Add dependency (Task A -> Task B) | N/A | 
| `buyruk project repair` | Rebuild `project.json` from `issues/` | N/A | 

## 5. LLM Optimization (L-SON)

**L-SON** (LLM-Simplified Object Notation) is designed for token efficiency in AI chats.

**Example View (`buyruk view PROJ-41 --format lson`):**

```text
@ID: PROJ-41
@TYPE: task
@STATUS: TODO
@PRIORITY: HIGH
@TITLE: Implement OAuth Flow
@DEP: PROJ-36
@PR: [github.com/pull/500](https://github.com/pull/500)
@DESC: Add Google and GitHub OAuth providers to the auth service.
```

## 6. Portability

* **Export:** Bundles a project folder into a single portable JSON file.
* **Import:** Reconstructs the local directory and index from an export file.