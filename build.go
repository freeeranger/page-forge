package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/yosssi/gohtml"
)

func BuildProject() {
	fmt.Println("Building project...")

	rootPath := "./test-site" // temporary, should just be . later

	if len(os.Args) != 2 {
		fmt.Println("Usage: page-forge build")
		return
	}

	if !validateProject() {
		return
	}

	// Clear out dir
	outDirPath := fmt.Sprintf("%s/out", rootPath)
	if _, err := os.Stat(outDirPath); !os.IsNotExist(err) {
		if err = os.RemoveAll(outDirPath); err != nil {
			fmt.Println("ERROR: Failed to clear out directory")
			return
		}
	}
	os.Mkdir(outDirPath, 0755)

	// Convert all files to html
	err := filepath.Walk(fmt.Sprintf("%s/pages", rootPath), func(path string, info os.FileInfo, err error) error {
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

		outputPath := fmt.Sprintf("%s\n", fmt.Sprintf("%s/out/%s", rootPath, fragments[len(fragments)-1]))
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

func convertFileToHTML(filePath string, outputPath string) {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("ERROR: Failed to read %s\n", filePath)
		return
	}

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(contents)

	str := gohtml.Format(UseTemplate("unnamed", outputPath, Traverse(doc)))

	file, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Println("Failed to write to html file")
		return
	}

	file.Truncate(0)
	file.Seek(0, 0)
	file.WriteString(str)
}

func UseTemplate(title string, path string, content string) string {
	rootPath := "./test-site/out/" // should be ./out/ later

	contents, err := os.ReadFile("res/default_template.html")
	if err != nil {
		fmt.Println("ERROR: Failed to open default template file")
		return ""
	}
	output := string(contents)
	output = strings.ReplaceAll(output, "{{PAGE-TITLE}}", title)
	output = strings.ReplaceAll(output, "{{CONTENT}}", content)

	config := ReadConfig()
	output = strings.ReplaceAll(output, "{{SITE-TITLE}}", config.Name)

	makeNavElement := func(title string, href string) string {
		return fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", href, title)
	}

	effectivePath := strings.TrimPrefix(path, rootPath)

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

func uniquePath(from string, to string) (string, error) {
	// Get the relative path
	rel, err := filepath.Rel(from, to)
	if err != nil {
		return "", err
	}

	// Check if the relative path is just ".." or deeper
	if strings.HasPrefix(rel, "..") {
		return rel, nil
	}

	// If it's a deeper path, return the unique part of `to`
	commonPath := filepath.Dir(from)
	uniquePart := strings.TrimPrefix(to, commonPath)
	uniquePart = strings.TrimPrefix(uniquePart, string(filepath.Separator)) // Remove leading separator

	return uniquePart, nil
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
	openingTag := ""
	closingTag := ""
	if tag != "" {
		openingTag = fmt.Sprintf("<%s>", tag)
		closingTag = fmt.Sprintf("</%s>", tag)
	}

	return InsertAdvancedHTMLElement(n, openingTag, closingTag)
}
