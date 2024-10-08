package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/yosssi/gohtml"
)

//go:embed res/default_template.html
var defaultTemplateContents string

func BuildProject() {
	fmt.Println("Building project...")

	if len(os.Args) != 2 {
		fmt.Println("Usage: page-forge build")
		return
	}

	if !validateProject() {
		return
	}

	// Clear out dir
	outDirPath := fmt.Sprintf("./out")
	if _, err := os.Stat(outDirPath); !os.IsNotExist(err) {
		if err = os.RemoveAll(outDirPath); err != nil {
			fmt.Println("ERROR: Failed to clear out directory")
			return
		}
	}
	os.Mkdir(outDirPath, 0755)

	// Convert all files to html
	err := filepath.Walk(fmt.Sprintf("./pages"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fragments := strings.Split(path, "pages/")

		// Create directories
		if info.IsDir() {
			outputPath := fmt.Sprintf("%s/%s", outDirPath, fragments[len(fragments)-1])
			os.Mkdir(outputPath, 0755)
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		outputPath := fmt.Sprintf("%s\n", fmt.Sprintf("./out/%s", fragments[len(fragments)-1]))
		outputPath = outputPath[:len(outputPath)-4] + ".html"
		convertFileToHTML(path, outputPath)

		return nil
	})

	if err != nil {
		fmt.Println("ERROR: Failed to iterate through all files in the pages directory")
		return
	}

	fmt.Println("Project successfully built, see out/")
}

func convertFileToHTML(inputPath string, outputPath string) {
	contents, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("ERROR: Failed to read %s\n", inputPath)
		return
	}

	// parse metadata and remove it from the contents string
	var metadata []MetadataEntry
	if strings.TrimSpace(string(contents[:4])) == "---" {
		charsToRemove := 4
		lines := strings.Split(string(contents), "\n")

		foundEnd := false
		for i := 1; i < len(lines); i++ {
			trimmedLine := strings.TrimSpace(lines[i])
			if foundEnd {
				if trimmedLine == "" { // remove all empty lines after the closing of the metadata
					charsToRemove += 1
					continue
				}
				break
			}

			charsToRemove += len(lines[i]) + 1

			if trimmedLine == "---" { // found the second --- (closing metadata section)
				foundEnd = true
				continue
			}

			// parse a line of metadata
			if strings.Count(lines[i], ":") == 0 {
				fmt.Println("ERROR: Incorrect metadata key, no value found")
				return
			}

			// temporary naive metadata key/value parsing
			segs := strings.SplitN(lines[i], ":", 2)
			metadata = append(metadata, MetadataEntry{strings.TrimSpace(segs[0]), strings.TrimSpace(segs[1])})
		}
		contents = contents[charsToRemove:]
	}

	_, pageTitle := filepath.Split(inputPath)
	pageTitle = pageTitle[:len(pageTitle)-3]

	pageSubtitle := ""

	// Do stuff with the metadata here
	if len(metadata) != 0 {
		for i := 0; i < len(metadata); i++ {
			if metadata[i].key == "title" {
				pageTitle = metadata[i].value
			} else if metadata[i].key == "subtitle" {
				pageSubtitle = metadata[i].value
			}
		}
	}

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(contents)

	str := gohtml.Format(UseTemplate(pageTitle, pageSubtitle, outputPath, Traverse(doc)))

	file, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Println("ERROR: Failed to write to html file")
		return
	}

	file.Truncate(0)
	file.Seek(0, 0)
	file.WriteString(str)
}

func UseTemplate(title string, subtitle string, path string, content string) string {
	config := ReadConfig()

	themePath := ""

	output := ""
	switch config.Theme {
	case "default":
		output = defaultTemplateContents
	default:
		themePath = fmt.Sprintf("./themes/%s", config.Theme)

		contents, err := os.ReadFile(themePath)
		if err != nil {
			fmt.Printf("ERROR: Failed to open template file %s\n", config.Theme)
			return ""
		}
		output = string(contents)
	}

	output = strings.ReplaceAll(output, "{{PAGE-TITLE}}", title)
	output = strings.ReplaceAll(output, "{{PAGE-SUBTITLE}}", subtitle)
	output = strings.ReplaceAll(output, "{{CONTENT}}", content)
	output = strings.ReplaceAll(output, "{{SITE-TITLE}}", config.Name)

	makeNavElement := func(title string, href string) string {
		return fmt.Sprintf("<li><a href=\"%s\" class=\"%s\">%s</a></li>", href, If(href == "", "nav-active", ""), title)
	}

	effectivePath := strings.TrimPrefix(path, "./out/")

	navElementString := ""
	for i := 0; i < len(config.NavElements); i++ {
		// Get the relative path from the current file to the one in the nav element
		relativeHref, _ := filepath.Rel(effectivePath, config.NavElements[i].Href)
		relativeHref = strings.TrimPrefix(relativeHref, "../")
		relativeHref = strings.TrimSuffix(relativeHref, ".")

		navElementString += makeNavElement(config.NavElements[i].Title, relativeHref)
	}
	output = strings.ReplaceAll(output, "{{NAV-ELEMENTS}}", navElementString)

	return output
}

func Traverse(node ast.Node) string {
	var buf bytes.Buffer

	switch n := node.(type) {
	case *ast.Text:
		if _, ok := n.Parent.GetParent().(*ast.BlockQuote); ok {
			buf.WriteString(fmt.Sprintf("<p>%s</p>", string(n.Literal))) // text in blockquotes need each line to be in a p tag
		} else {
			buf.WriteString(string(n.Literal)) // all other text elements
		}

	case *ast.Heading:
		buf.WriteString(InsertHTMLElement(n, fmt.Sprintf("h%d", n.Level)))

	case *ast.Container:
		buf.WriteString(InsertHTMLElement(n, "div"))

	case *ast.Link:
		// TODO need to calculate the relative link for the page
		buf.WriteString(InsertAdvancedHTMLElement(n, fmt.Sprintf("<a href=\"%s\">", string(n.Destination)), "</a>"))

	case *ast.List:
		el := ""
		if n.ListFlags == 16 {
			el = "ul"
		} else if n.ListFlags == 17 {
			el = "ol"
		}

		if el != "" {
			buf.WriteString(InsertHTMLElement(n, el))
		}

	case *ast.ListItem:
		buf.WriteString(InsertHTMLElement(n, "li"))

	case *ast.CodeBlock:
		buf.WriteString(fmt.Sprintf("<pre><code class=\"code-block language-%s\">", string(n.Info)) + string(n.Literal) + "</code></pre>")

	case *ast.Code:
		buf.WriteString("<span class=\"inline-code\">" + string(n.Literal) + "</span>")

	case *ast.BlockQuote:
		buf.WriteString(InsertAdvancedHTMLElement(n, "<div class=\"block-quote\">", "</div>"))

	case *ast.Strong:
		buf.WriteString(InsertAdvancedHTMLElement(n, "<span style=\"font-weight: bold;\">", "</span>"))

	case *ast.Emph:
		buf.WriteString(InsertAdvancedHTMLElement(n, "<span style=\"font-style: italic;\">", "</span>"))

	case *ast.Paragraph:
		buf.WriteString(InsertHTMLElement(n, "p"))

	default:
		buf.WriteString(InsertHTMLElement(n, ""))
	}

	return buf.String()
}

func InsertAdvancedHTMLElement(n ast.Node, openingTag string, closingTag string) string {
	var buf bytes.Buffer

	buf.WriteString(openingTag)
	for i := 0; i < len(n.GetChildren()); i++ {
		buf.WriteString(Traverse(n.GetChildren()[i]))
	}
	buf.WriteString(closingTag)

	return buf.String()
}

func InsertHTMLElement(n ast.Node, tag string) string {
	openingTag := If(tag == "", "", fmt.Sprintf("<%s>", tag))
	closingTag := If(tag == "", "", fmt.Sprintf("</%s>", tag))

	return InsertAdvancedHTMLElement(n, openingTag, closingTag)
}
