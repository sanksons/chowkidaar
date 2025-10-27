# 🔐 Chowkidaar (चौकीदार)
### *Your Faithful Password Guardian*

<div align="center">

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Linux%20|%20macOS%20|%20Windows-lightgrey?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Security](https://img.shields.io/badge/Security-AES--256--GCM%20|%20Argon2id-red?style=for-the-badge)

*चौकीदार आपके पासवर्ड का रक्षक है*

</div>

---

## 🏛️ What is Chowkidaar?

**Chowkidaar** (meaning "watchman" or "guardian" in Hindi/Urdu) is a modern, secure command-line password manager inspired by the Unix `pass` utility. Just like a faithful watchman guards your home, Chowkidaar vigilantly protects your digital secrets with military-grade encryption and seamless synchronization across all your devices.

### 🎯 Philosophy

> *"A good chowkidaar never sleeps, never forgets, and always keeps your secrets safe."*

Chowkidaar embodies the principles of:
- **🛡️ Security First**: Military-grade encryption (Argon2id + AES-256-GCM)
- **🌳 Organized**: Hierarchical password storage like your file system
- **🔄 Synchronized**: Git-powered multi-device sync
- **🚀 Fast**: Smart caching for seamless workflow
- **🎨 Beautiful**: Interactive tree navigation and elegant CLI

---

## ✨ Features That Make Chowkidaar Special

### 🔒 **Bank-Vault Level Security**
- **Argon2id Key Derivation**: OWASP-recommended password hashing
- **AES-256-GCM Encryption**: Military-grade symmetric encryption
- **Secure Memory Handling**: No plaintext passwords in memory dumps
- **Master Password Protection**: Single key to rule them all

### 🔄 **Git-Powered Synchronization**
- **Multi-Device Sync**: Access passwords anywhere
- **Version History**: Never lose a password again
- **Conflict Resolution**: Smart merging across devices
- **Team Sharing**: Share password stores securely
- **Auto-Commit**: Automatic backup of changes

### 🧠 **Smart Caching System**
- **Secure Password Cache**: Encrypted in-memory storage
- **Session Management**: Per-process cache isolation
- **Configurable Timeouts**: Balance security and convenience
- **Cross-Process Safety**: Multiple instances work seamlessly

### 🎨 **Developer Experience**
- **Intuitive Commands**: Natural language-like interface
- **Rich Help System**: Built-in documentation and examples
- **Error Recovery**: Helpful error messages and suggestions
- **Shell Integration**: Perfect for scripts and automation

---

## 🚀 Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/sanksons/chowkidaar.git
cd chowkidaar
go build -o chowkidaar .

# Move to your PATH (optional)
sudo mv chowkidaar /usr/local/bin/
```

### Initialize Your Password Vault

```bash
# Local password store
chowkidaar init

# With Git synchronization (recommended)
chowkidaar init --git-url git@github.com:username/passwords.git
```

### Your First Password

```bash
# Add a password
chowkidaar insert Personal/github

# List passwords with tree view
chowkidaar list                # Tree view

# Retrieve a password  
chowkidaar show Personal/github

# Edit existing password
chowkidaar edit Personal/github
```

---

## 📚 Command Reference

### Core Commands

```bash
# Initialize password store
chowkidaar init [--git-url <url>]

# Password management
chowkidaar insert <name>      # Add new password
chowkidaar show <name>        # Show password
chowkidaar edit <name>        # Edit password
chowkidaar remove <name>      # Delete password
chowkidaar list [subfolder]   # List passwords
```

### Git Synchronization

```bash
# Git operations
chowkidaar git status         # Check repository status
chowkidaar git push           # Push changes to remote
chowkidaar git pull           # Pull changes from remote  
chowkidaar git sync           # Full synchronization (pull + push)
```

### Cache Management

```bash
# Cache operations
chowkidaar cache status       # Show cache status
chowkidaar cache clear        # Clear cached passwords
chowkidaar cache timeout 10   # Set cache timeout (minutes)
```

---

## 🔧 Configuration

### Environment Variables

```bash
# Core settings
export PASSWORD_STORE_DIR="$HOME/.password-store"
export PASSWORD_STORE_CACHE_TIMEOUT=5  # minutes
export EDITOR="vim"  # or nano, code, etc.

# Git integration
export PASSWORD_STORE_GIT_URL="git@github.com:username/passwords.git"
export PASSWORD_STORE_GIT_AUTO_SYNC=true

# Authentication (for HTTPS)
export GIT_USERNAME="your-username"
export GIT_TOKEN="your-personal-access-token"
```

### Secure Git Authentication

#### SSH Keys (Recommended)
```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "your.email@example.com"

# Add to SSH agent
ssh-add ~/.ssh/id_ed25519

# Use with chowkidaar
chowkidaar init --git-url git@github.com:username/passwords.git
```

#### HTTPS with .netrc
```bash
# Create ~/.netrc file
cat >> ~/.netrc << EOF
machine github.com
login your-username
password your-token
EOF
chmod 600 ~/.netrc
```

---

## 🏗️ Architecture & Security

### Encryption Stack

```
┌─────────────────────────┐
│     Your Password       │
├─────────────────────────┤
│   AES-256-GCM Cipher    │  ← Authenticated Encryption
├─────────────────────────┤
│   Argon2id Key Derive   │  ← OWASP Recommended
├─────────────────────────┤
│   Master Password       │  ← You remember this
└─────────────────────────┘
```

### File Structure

```
~/.password-store/
├── .git/                   # Git repository
├── .cache/                 # Encrypted cache (auto-created)
├── .master                 # Master password hash
├── .git-config            # Git sync configuration
├── Work/
│   ├── email.enc          # Encrypted password files
│   └── servers/
│       ├── production.enc
│       └── staging.enc
└── Personal/
    ├── bank.enc
    └── social/
        ├── twitter.enc
        └── facebook.enc
```

### Security Features

- **🔐 Zero-Knowledge Architecture**: Only you know your master password
- **🧂 Unique Salts**: Each password uses a unique salt
- **⏱️ Time-Based Cache**: Configurable cache expiration
- **🔄 Session Isolation**: Cache tied to specific sessions
- **🛡️ Memory Protection**: Sensitive data cleared from memory
- **📝 Audit Trail**: Git history tracks all changes

---

## 🌟 Advanced Usage

### Multi-Device Workflow

**Device 1 (Setup)**
```bash
# Initialize with Git
chowkidaar init --git-url git@github.com:username/passwords.git

# Add some passwords
chowkidaar insert Work/email
chowkidaar insert Personal/bank
```

**Device 2 (Sync)**
```bash
# Clone existing password store
chowkidaar init --git-url git@github.com:username/passwords.git

# Passwords are automatically available!
chowkidaar list     # View password tree
```

### Team Password Sharing

```bash
# Create shared repository
chowkidaar init --git-url git@github.com:team/shared-passwords.git

# Team members clone the same repository
# Each person uses their own master password for local encryption
```

### Scripting & Automation

```bash
#!/bin/bash
# Backup script example

# Check if master password is cached
if chowkidaar cache status | grep -q "cached"; then
    # Perform bulk operations without password prompts
    chowkidaar git sync
    echo "Passwords synchronized!"
else
    echo "Master password required for sync"
fi
```

---

## 🔮 Roadmap & Future Plans

### 🚀 Upcoming Features
- [ ] **Browser Integration**: Native browser extensions
- [ ] **Mobile Apps**: iOS and Android companion apps  
- [ ] **GUI Client**: Cross-platform desktop application
- [ ] **Password Generator**: Built-in secure password generation
- [ ] **Two-Factor Auth**: TOTP integration
- [ ] **Secure Sharing**: Time-limited password sharing links
- [ ] **Import/Export**: Support for major password managers
- [ ] **Templates**: Password templates for common services

### 🎨 CLI Enhancements
- [ ] **File Preview**: Show password strength and metadata
- [ ] **Search**: Real-time password search functionality
- [ ] **Bulk Operations**: Select multiple passwords for operations
- [ ] **Themes**: Customizable color schemes
- [ ] **Better Navigation**: Enhanced tree view experience

---

## 🤝 Contributing

We welcome contributions! Chowkidaar is built with love for the community.

### Development Setup
```bash
# Clone repository
git clone https://github.com/sanksons/chowkidaar.git
cd chowkidaar

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build
go build -o chowkidaar .
```

### Areas for Contribution
- 🐛 **Bug Fixes**: Help make Chowkidaar more reliable
- ✨ **Features**: Implement items from our roadmap
- 📚 **Documentation**: Improve guides and examples
- 🎨 **UI/UX**: Enhance the interactive experience
- 🔒 **Security**: Security audits and improvements
- 🌍 **Localization**: Multi-language support

---

## 📖 Inspiration & Philosophy

Chowkidaar draws inspiration from:

- **Unix `pass`**: The standard Unix password manager
- **Git Philosophy**: Distributed version control for passwords
- **Vim Navigation**: Efficient keyboard-driven interfaces
- **Modern CLI Tools**: Beautiful, intuitive command-line experiences

### Why "Chowkidaar"?

In Indian culture, a *chowkidaar* is a trusted guardian who watches over your property with unwavering dedication. They're always alert, completely reliable, and deeply respected for their service. This perfectly embodies what we want in a password manager:

- **🛡️ Always Vigilant**: Constantly protecting your secrets
- **🤝 Completely Trusted**: You can rely on it with your most sensitive data
- **🏠 Part of the Family**: Seamlessly integrated into your daily workflow
- **🌙 Never Sleeps**: Available whenever you need it

---

## ⚠️ Security Considerations

### What Chowkidaar Protects Against
- ✅ **Data Breaches**: All passwords encrypted locally
- ✅ **Shoulder Surfing**: No passwords displayed in logs
- ✅ **Memory Dumps**: Sensitive data cleared from memory
- ✅ **Offline Attacks**: Strong key derivation (Argon2id)
- ✅ **Network Interception**: Git over SSH/HTTPS

### What You Must Protect
- 🔑 **Master Password**: This is your single point of failure
- 🖥️ **Local Device**: Ensure physical security
- 🔐 **Git Repository**: Keep your Git credentials secure
- 💾 **Backups**: Secure your Git repository backups

### Best Practices
1. **Strong Master Password**: Use a unique, complex master password
2. **Regular Sync**: Frequently sync to avoid data loss
3. **Private Repositories**: Never use public Git repositories
4. **Device Security**: Keep your devices updated and secure
5. **Access Control**: Limit Git repository access appropriately

---

## 🆘 Support & Community

### Getting Help
- 📖 **Documentation**: Check this README and built-in help (`chowkidaar --help`)
- 🐛 **Issues**: Report bugs on [GitHub Issues](https://github.com/sanksons/chowkidaar/issues)
- 💡 **Feature Requests**: Suggest features in GitHub Discussions
- 🔧 **Troubleshooting**: See common issues in our Wiki

### Community Resources
- 🌐 **Website**: [Project Homepage](https://github.com/sanksons/chowkidaar)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/sanksons/chowkidaar/discussions)
- 📝 **Blog**: Development updates and tutorials
- 🎥 **Videos**: Setup guides and demonstrations

---

## 📜 License

Chowkidaar is released under the **MIT License**. See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

Special thanks to:
- **Jason A. Donenfeld** - Creator of Unix `pass`, the inspiration for this project
- **The Go Community** - For excellent cryptographic libraries
- **Git Developers** - For the foundation of our sync system
- **Security Researchers** - For establishing best practices we follow

---

<div align="center">

### *"Your secrets are safe with your faithful chowkidaar."*

**[⭐ Star us on GitHub](https://github.com/sanksons/chowkidaar)** • **[🚀 Get Started](#-quick-start)** • **[🤝 Contribute](#-contributing)**

---

*Made with ❤️ by developers who believe security should be beautiful and simple.*

</div>