package oambackend

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

type Formatter struct {
	writer io.Writer
}

func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{writer: w}
}

func (f *Formatter) RenderTable(title string, headers []string, rows [][]string) {
	if title != "" {
		fmt.Fprintln(f.writer, title)
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, strings.Join(headers, "\t"))

	seps := make([]string, len(headers))
	for i, h := range headers {
		seps[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(w, strings.Join(seps, "\t"))

	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	w.Flush()
}

func (f *Formatter) WriteCSV(filePath string, headers []string, rows [][]string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	if err := w.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func ResolveFilename(pathFlag, context, cmd string) string {
	if pathFlag == "auto" {
		return time.Now().Format("20060102-150405") + "-" + context + "-" + cmd + ".csv"
	}
	return pathFlag
}
