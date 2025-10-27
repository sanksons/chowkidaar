# Git Integration

The password manager supports Git synchronization similar to the Unix `pass` password manager, allowing you to sync your encrypted passwords across multiple devices and maintain version history.

## Key Features

- **Automatic cloning** of existing password stores from remote repositories
- **Auto-sync** commits and pushes changes automatically when passwords are added/removed
- **Manual sync commands** for push, pull, and status operations
- **Cross-device synchronization** using any Git hosting service (GitHub, GitLab, etc.)
- **Version history** tracking of all password changes

## Quick Start

### Initialize with Git Sync

```bash
# Initialize new password store with Git sync
chowkidaar init --git-url https://github.com/username/passwords.git

# Or clone existing password store
chowkidaar init --git-url https://github.com/username/existing-passwords.git
```

### Environment Variables

```bash
# Set Git URL for automatic sync
export PASSWORD_STORE_GIT_URL="https://github.com/username/passwords.git"

# Enable/disable auto-sync (default: true)
export PASSWORD_STORE_GIT_AUTO_SYNC=true

# Then initialize normally
chowkidaar init
```

## How It Works

### Initialization Scenarios

1. **New Repository**: Creates local store, initializes Git, sets up remote
2. **Empty Remote**: Creates local store, pushes initial structure to remote
3. **Existing Passwords**: Clones remote repository with existing encrypted passwords

### Automatic Synchronization

When auto-sync is enabled (default), the following operations automatically commit and push:

- `chowkidaar insert <name>` - Commits "Add password for <name>"
- `chowkidaar remove <name>` - Commits "Remove password for <name>"
- `chowkidaar edit <name>` - Commits "Update password for <name>"

### File Structure in Git

```
.password-store/
├── .git/                    # Git repository
├── .gitignore              # Excludes cache and temporary files
├── .master                 # Encrypted master password hash
├── Email/
│   ├── gmail.com.enc       # Encrypted password files
│   └── work.enc
└── Social/
    ├── facebook.enc
    └── twitter.enc
```

## Git Commands

### Status
```bash
# Check Git repository status
chowkidaar git status
```
Shows modified, added, and deleted password files.

### Push
```bash
# Push local changes to remote
chowkidaar git push
```
Commits any local changes and pushes to the remote repository.

### Pull
```bash
# Pull changes from remote
chowkidaar git pull
```
Pulls and merges remote changes into local store.

### Sync
```bash
# Full synchronization (recommended)
chowkidaar git sync
```
Performs pull → commit local changes → push sequence for complete sync.

## Authentication Setup

Chowkidaar supports multiple authentication methods:

### SSH Keys (Recommended)
```bash
# Setup SSH key (if not already done)
ssh-keygen -t ed25519 -C "your.email@example.com"

# Add to SSH agent
ssh-add ~/.ssh/id_ed25519

# Add public key to your Git service (GitHub, GitLab, etc.)
cat ~/.ssh/id_ed25519.pub

# Use SSH URL
chowkidaar init --git-url git@github.com:username/passwords.git
```

### HTTPS with Credentials
```bash
# Method 1: .netrc file (Recommended for HTTPS)
# Create ~/.netrc file with:
echo "machine github.com login your-username password your-token" >> ~/.netrc
echo "machine gecgithub01.walmart.com login your-corp-username password your-corp-token" >> ~/.netrc
chmod 600 ~/.netrc
chowkidaar init --git-url https://github.com/username/passwords.git

# Method 2: Environment variables
export GIT_USERNAME="your-username"
export GIT_TOKEN="your-personal-access-token"
chowkidaar init --git-url https://github.com/username/passwords.git

# Method 3: Embedded in URL
chowkidaar init --git-url https://username:token@github.com/username/passwords.git

# Method 4: Interactive prompt (will ask for credentials)
chowkidaar init --git-url https://github.com/username/passwords.git
```

### .netrc File Format

Chowkidaar automatically reads credentials from `~/.netrc` (or `~/_netrc` on Windows):

```bash
# Example .netrc file
machine github.com
login your-github-username
password ghp_your_personal_access_token

machine gitlab.com  
login your-gitlab-username
password glpat-your_gitlab_token

machine gecgithub01.walmart.com
login your-walmart-username
password your-walmart-token

# Default entry for any unspecified hosts
default
login fallback-username
password fallback-password
```

**Security Note**: Always set proper permissions:
```bash
chmod 600 ~/.netrc  # Owner read/write only
```

## Setup Examples

### GitHub Setup

1. **Create Private Repository** on GitHub (recommended for passwords)
2. **Generate Personal Access Token** or setup SSH keys
3. **Initialize password store**:

```bash
# Using HTTPS with token
chowkidaar init --git-url https://token@github.com/username/passwords.git

# Using SSH (recommended)
chowkidaar init --git-url git@github.com:username/passwords.git
```

### GitLab Setup

```bash
# Private GitLab repository
chowkidaar init --git-url https://gitlab.com/username/passwords.git
```

### Enterprise Git (like Walmart's gecgithub01)

```bash
# For enterprise Git servers, use your corporate credentials
export GIT_USERNAME="your-corp-username"
export GIT_TOKEN="your-corp-token"
chowkidaar init --git-url https://gecgithub01.walmart.com/username/passwords.git
```

### Self-hosted Git

```bash
# Your own Git server
chowkidaar init --git-url https://git.yourserver.com/passwords.git
```

## Multi-Device Workflow

### Initial Setup on Device 1
```bash
# Create and setup password store
chowkidaar init --git-url git@github.com:username/passwords.git
chowkidaar insert Email/gmail
chowkidaar insert Social/github
```

### Setup on Device 2
```bash
# Clone existing password store
chowkidaar init --git-url git@github.com:username/passwords.git
# Existing passwords are now available!
chowkidaar list
```

### Daily Usage
```bash
# On any device - add/modify passwords
chowkidaar insert Work/newsite
chowkidaar edit Social/github

# Changes are automatically synced!
# On other devices, just pull updates:
chowkidaar git pull
```

## Configuration

### Auto-sync Settings
```bash
# Disable auto-sync for manual control
export PASSWORD_STORE_GIT_AUTO_SYNC=false

# Enable auto-sync (default)
export PASSWORD_STORE_GIT_AUTO_SYNC=true
```

### Manual Sync Workflow
```bash
# When auto-sync is disabled
chowkidaar insert newpassword        # No auto-commit
chowkidaar git status                # Check changes
chowkidaar git push                  # Manual commit & push
```

## Security Considerations

1. **Use Private Repositories** - Never use public repos for passwords
2. **SSH Keys Recommended** - More secure than HTTPS with tokens
3. **Two-Factor Authentication** - Enable 2FA on your Git hosting service
4. **Repository Access** - Limit access to trusted devices only
5. **Password Protection** - Your master password is still the primary security layer

## Troubleshooting

### Authentication Issues

#### SSH Authentication
```bash
# Test SSH connection
ssh -T git@github.com
ssh -T git@gecgithub01.walmart.com  # For enterprise

# Check SSH agent
ssh-add -l

# Add SSH key if missing
ssh-add ~/.ssh/id_ed25519
```

#### HTTPS Authentication
```bash
# Check .netrc file
cat ~/.netrc
ls -la ~/.netrc  # Should show 600 permissions

# Verify environment credentials
echo $GIT_USERNAME
echo $GIT_TOKEN

# Test .netrc parsing manually
curl -n -s https://api.github.com/user  # Should authenticate via .netrc

# Generate personal access token with repo permissions:
# GitHub: Settings > Developer settings > Personal access tokens
# GitLab: User Settings > Access Tokens  
# Enterprise Git: Check with your IT team
```

#### Corporate Networks
```bash
# For corporate networks, you might need:
# 1. VPN connection
# 2. Corporate credentials instead of personal GitHub account
# 3. Special enterprise Git URLs

# Example .netrc setup for Walmart:
cat >> ~/.netrc << EOF
machine gecgithub01.walmart.com
login s1b0432
password your-walmart-git-token
EOF
chmod 600 ~/.netrc

# Then simply run:
chowkidaar init --git-url https://gecgithub01.walmart.com/s1b0432/testrepo.git

# Alternative: Environment variables
export GIT_USERNAME="s1b0432"
export GIT_TOKEN="your-walmart-git-token"
chowkidaar init --git-url https://gecgithub01.walmart.com/s1b0432/testrepo.git
```

### Merge Conflicts
The password manager avoids conflicts by using file-per-password structure. If conflicts occur:
```bash
# Check status
chowkidaar git status

# Manual resolution may be needed
# Use standard Git commands in ~/.password-store/
cd ~/.password-store
git status
git add .
git commit -m "Resolve conflicts"
git push
```

### Reset Remote URL
```bash
# Change remote repository
cd ~/.password-store
git remote set-url origin new-repo-url
```

## Integration with `pass`

The Git structure is compatible with Unix `pass`, so you can:
- Import existing `pass` stores by cloning their Git repositories
- Use `pass` commands on other systems with the same repository
- Migrate from `pass` to `chowkidaar` seamlessly

## Best Practices

1. **Regular Sync**: Use `chowkidaar git sync` regularly on all devices
2. **Private Repos**: Always use private repositories for security
3. **Backup Strategy**: Git serves as your backup - keep remote repository secure
4. **Access Control**: Use SSH keys and limit repository access
5. **Commit Messages**: Auto-generated messages provide clear audit trail