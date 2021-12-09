package main

import "testing"

func TestCleanupMarkDown(t *testing.T) {
	tests := []struct {
		doc, in, expected string
	}{
		{
			doc: "whitespace around sections",
			in: `

	## Section start

Some lines.
And more lines.

`,
			expected: `## Section start

Some lines.
And more lines.`,
		},
		{
			doc: "lines with inline tabs",
			in: `## Some	Heading

A line with tabs		in it.
Tabs	should be replaced by spaces`,
			expected: `## Some    Heading

A line with tabs        in it.
Tabs    should be replaced by spaces`,
		},
		{
			doc: "lines with trailing spaces",
			in: `## Some Heading with spaces                  
       
This is a line.              
    This is an indented line        

### Some other heading         

Last line.`,
			expected: `## Some Heading with spaces

This is a line.
    This is an indented line

### Some other heading

Last line.`,
		},
		{
			doc: "lines with trailing tabs",
			in: `## Some Heading with tabs				
		
This is a line.		
	This is an indented line		

### Some other heading 	

Last line.`,
			expected: `## Some Heading with tabs

This is a line.
    This is an indented line

### Some other heading

Last line.`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			out, _ := cleanupMarkDown(tc.in)
			if out != tc.expected {
				t.Fatalf("\nexpected:\n%q\nactual:\n%q\n", tc.expected, out)
			}
		})
	}
}

func TestConvertHTMLAnchor(t *testing.T) {
	tests := []struct {
		in, id, expected string
	}{
		{
			in:       `# <a name=heading1></a> Heading 1`,
			id:       "heading1",
			expected: `# Heading 1 {#heading1}`,
		},
		{
			in:       `## Heading 2<a name=heading2></a> `,
			id:       "heading2",
			expected: `## Heading 2 {#heading2}`,
		},
		{
			in:       `### <a id=heading3></a>Heading 3`,
			id:       "heading3",
			expected: `### Heading 3 {#heading3}`,
		},
		{
			in:       `#### <a id="heading4"></a> Heading 4`,
			id:       "heading4",
			expected: `#### Heading 4 {#heading4}`,
		},
		{
			in:       `##### <a   id="heading5"  ></a>  Heading 5`,
			id:       "heading5",
			expected: `##### Heading 5 {#heading5}`,
		},
		{
			in:       `###### <a id=hello href=foo>hello!</a>Heading 6`,
			id:       "",
			expected: `###### <a id=hello href=foo>hello!</a>Heading 6`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			out, id := convertHTMLAnchor(tc.in)
			if id != tc.id {
				t.Fatalf("expected: %s, actual:   %s\n", tc.id, id)
			}
			if out != tc.expected {
				t.Fatalf("\nexpected: %s\nactual:   %s\n", tc.expected, out)
			}
		})
	}
}
