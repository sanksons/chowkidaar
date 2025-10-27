package list

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ListOptions holds configuration for list display
type ListOptions struct {
	ShowIcons    bool
	ShowColors   bool
	Flat         bool
	ShowDetails  bool
	MaxDepth     int
	SearchFilter string
}

// DefaultOptions returns sensible default list options
func DefaultOptions() *ListOptions {
	return &ListOptions{
		ShowIcons:   true,
		ShowColors:  true,
		Flat:        false,
		ShowDetails: false,
		MaxDepth:    -1, // No limit
	}
}

// Entry represents a password store entry
type Entry struct {
	Name        string
	Path        string
	IsDirectory bool
	Size        int64
	ModTime     time.Time
	Children    []*Entry
	Depth       int
}

// ListBuilder builds and displays password store listings
type ListBuilder struct {
	baseDir string
	options *ListOptions
}

// NewListBuilder creates a new list builder
func NewListBuilder(baseDir string, options *ListOptions) *ListBuilder {
	if options == nil {
		options = DefaultOptions()
	}
	return &ListBuilder{
		baseDir: baseDir,
		options: options,
	}
}

// Generate creates the entry tree and displays it
func (lb *ListBuilder) Generate(subfolder string) error {
	searchDir := lb.baseDir
	if subfolder != "" {
		searchDir = filepath.Join(lb.baseDir, subfolder)
	}

	// Check if directory exists
	if _, err := os.Stat(searchDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", searchDir)
	}

	// Build entry tree
	root, err := lb.buildTree(searchDir, "", 0)
	if err != nil {
		return fmt.Errorf("failed to build directory tree: %w", err)
	}

	// Check if we have any entries
	if len(root.Children) == 0 {
		if lb.options.ShowColors {
			fmt.Println("🔐 \033[33mNo passwords found in this directory.\033[0m")
			fmt.Println("   Use '\033[32mchowkidaar insert <name>\033[0m' to add passwords.")
		} else {
			fmt.Println("No passwords found in this directory.")
			fmt.Println("Use 'chowkidaar insert <name>' to add passwords.")
		}
		return nil
	}

	// Show header for tree view
	if !lb.options.Flat {
		if lb.options.ShowColors {
			if subfolder != "" {
				fmt.Printf("🔐 \033[1m%s\033[0m\n", subfolder)
			} else {
				fmt.Printf("🔐 \033[1mPassword Store\033[0m\n")
			}
		} else {
			if subfolder != "" {
				fmt.Printf("Password Store: %s\n", subfolder)
			} else {
				fmt.Printf("Password Store\n")
			}
		}
	}

	// Display the tree
	if lb.options.Flat {
		return lb.displayFlat(root)
	}
	return lb.displayTree(root)
}

// buildTree recursively builds the entry tree
func (lb *ListBuilder) buildTree(dir, relativePath string, depth int) (*Entry, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Name:        filepath.Base(dir),
		Path:        relativePath,
		IsDirectory: info.IsDir(),
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		Depth:       depth,
	}

	// Stop if we've reached max depth
	if lb.options.MaxDepth >= 0 && depth >= lb.options.MaxDepth {
		return entry, nil
	}

	if info.IsDir() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return entry, nil // Don't fail completely, just skip this directory
		}

		// Sort entries: directories first, then files, both alphabetically
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})

		for _, childEntry := range entries {
			// Skip hidden files and directories starting with .
			if strings.HasPrefix(childEntry.Name(), ".") {
				continue
			}

			childPath := filepath.Join(dir, childEntry.Name())
			childRelativePath := filepath.Join(relativePath, childEntry.Name())

			child, err := lb.buildTree(childPath, childRelativePath, depth+1)
			if err != nil {
				continue // Skip problematic entries
			}

			// Apply search filter if specified
			if lb.options.SearchFilter != "" {
				if lb.matchesFilter(child) {
					entry.Children = append(entry.Children, child)
				}
			} else {
				entry.Children = append(entry.Children, child)
			}
		}
	}

	return entry, nil
}

// matchesFilter checks if an entry matches the search filter
func (lb *ListBuilder) matchesFilter(entry *Entry) bool {
	filter := strings.ToLower(lb.options.SearchFilter)

	// Check if name matches
	if strings.Contains(strings.ToLower(entry.Name), filter) {
		return true
	}

	// Check if any child matches (for directories)
	for _, child := range entry.Children {
		if lb.matchesFilter(child) {
			return true
		}
	}

	return false
}

// displayTree displays entries in tree format
func (lb *ListBuilder) displayTree(root *Entry) error {
	if root.Depth == 0 {
		// Don't show root directory itself, just its children
		for i, child := range root.Children {
			isLast := i == len(root.Children)-1
			lb.printEntryWithLast(child, "", isLast)
		}
	} else {
		lb.printEntryWithLast(root, "", true)
	}
	return nil
}

// displayFlat displays entries in flat list format
func (lb *ListBuilder) displayFlat(root *Entry) error {
	var entries []*Entry
	lb.collectAllEntries(root, &entries)

	if len(entries) == 0 {
		fmt.Println("No passwords found.")
		return nil
	}

	// Print header if showing details
	if lb.options.ShowDetails {
		fmt.Printf("%-40s %10s %s\n", "Name", "Modified", "Path")
		fmt.Println(strings.Repeat("─", 70))
	}

	for _, entry := range entries {
		if !entry.IsDirectory {
			if lb.options.ShowDetails {
				modTime := entry.ModTime.Format("2006-01-02")
				name := strings.TrimSuffix(entry.Name, ".enc")
				fmt.Printf("%-40s %10s %s\n", name, modTime, entry.Path)
			} else {
				fmt.Println(lb.formatEntryName(entry))
			}
		}
	}
	return nil
}

// collectAllEntries recursively collects all entries for flat display
func (lb *ListBuilder) collectAllEntries(entry *Entry, entries *[]*Entry) {
	if entry.Depth > 0 { // Skip root
		*entries = append(*entries, entry)
	}

	for _, child := range entry.Children {
		lb.collectAllEntries(child, entries)
	}
}

// printEntryWithLast prints an entry with proper tree formatting
func (lb *ListBuilder) printEntryWithLast(entry *Entry, prefix string, isLast bool) {
	// Format the entry line
	line := lb.formatTreeLine(entry, prefix, isLast)
	fmt.Println(line)

	// Print children if it's a directory
	if entry.IsDirectory && len(entry.Children) > 0 {
		for i, child := range entry.Children {
			childIsLast := i == len(entry.Children)-1
			var newPrefix string

			if isLast {
				newPrefix = prefix + "    "
			} else {
				newPrefix = prefix + "│   "
			}

			lb.printEntryWithLast(child, newPrefix, childIsLast)
		}
	}
}

// formatTreeLine formats a single line for tree display
func (lb *ListBuilder) formatTreeLine(entry *Entry, prefix string, isLast bool) string {
	var line strings.Builder

	// Add prefix
	line.WriteString(prefix)

	// Add tree connector
	if entry.Depth > 0 {
		if isLast {
			line.WriteString("└── ")
		} else {
			line.WriteString("├── ")
		}
	}

	// Add icon and name
	line.WriteString(lb.formatEntryName(entry))

	// Add details if requested
	if lb.options.ShowDetails && !entry.IsDirectory {
		modTime := entry.ModTime.Format("Jan 02")
		if lb.options.ShowColors {
			line.WriteString(fmt.Sprintf(" \033[90m(%s)\033[0m", modTime))
		} else {
			line.WriteString(fmt.Sprintf(" (%s)", modTime))
		}
	}

	return line.String()
}

// formatEntryName formats the entry name with icons and colors
func (lb *ListBuilder) formatEntryName(entry *Entry) string {
	var name strings.Builder

	// Add icon
	if lb.options.ShowIcons {
		if entry.IsDirectory {
			if len(entry.Children) > 0 {
				name.WriteString("📂 ") // Open folder icon
			} else {
				name.WriteString("📁 ") // Closed folder icon
			}
		} else {
			name.WriteString("🔑 ") // Key icon for passwords
		}
	} else {
		// Text-based icons for terminals without emoji support
		if entry.IsDirectory {
			name.WriteString("[DIR] ")
		} else {
			name.WriteString("[PWD] ")
		}
	}

	// Clean up name (remove .enc extension)
	displayName := entry.Name
	if !entry.IsDirectory {
		displayName = strings.TrimSuffix(displayName, ".enc")
	}

	// Add color coding
	if lb.options.ShowColors {
		if entry.IsDirectory {
			name.WriteString(fmt.Sprintf("\033[1;34m%s\033[0m", displayName)) // Bold Blue for directories
		} else {
			name.WriteString(fmt.Sprintf("\033[32m%s\033[0m", displayName)) // Green for passwords
		}
	} else {
		name.WriteString(displayName)
	}

	return name.String()
}

// GenerateWithOptions provides a convenient way to generate listings with custom options
func GenerateWithOptions(baseDir, subfolder string, options *ListOptions) error {
	builder := NewListBuilder(baseDir, options)
	return builder.Generate(subfolder)
}
