# JSON Structure Quick Reference

Document parsing response structure (e.g., from Marker, PyMuPDF, or similar PDF parsers).

## Root Structure
```json
[ {DocumentObject} ]
```
Array of processed documents, supporting multiple files in a single response.

## Document Object - All Fields

| Path | Type | Description |
|------|------|-------------|
| `content` | string | Full extracted text content |
| `document` | object | Document-level metadata/structure |
| `elements` | array | Semantic chunks (titles, text, etc.) |
| `images` | array | Embedded images (raw byte arrays) |
| `metadata` | object | Document metadata |
| `mime_type` | string | "application/pdf" |
| `pages` | array | Page-level content and details |
| `quality_score` | number | Parsing confidence (0.0-1.0) |
| `tables` | array\|null | Extracted table data |
| `document.nodes` | array | Graph structure for navigation |

## metadata.*
```json
{
  "title": "string",
  "authors": ["string"],
  "created_by": "string",
  "format_type": "string",
  "pdf_version": "string",
  "producer": "string",
  "is_encrypted": boolean,
  "width": number,
  "height": number,
  "page_count": number,
  "output_format": "string",
  "quality_score": number,
  "pages": {
    "total_count": number,
    "unit_type": "string",
    "boundaries": [{"byte_start": number, "byte_end": number}],
    "pages": [...]
  }
}
```

## images[*]
```json
{
  "data": [number, ...],     // RGBA/RGB pixel values (raw bytes)
  "format": "string",        // Image format
  "image_index": number,
  "page_number": number,
  "width": number,
  "height": number,
  "colorspace": "string",
  "bits_per_component": number,
  "is_mask": boolean
}
```

## pages[*]
```json
{
  "page_number": number,
  "content": "string",       // Page-specific text content
  "images": [...],           // Page-specific images
  "is_blank": boolean,
  "hierarchy": {
    "block_count": number,
    "blocks": [...]
  }
}
```

## elements[*]
```json
{
  "element_id": "string",    // Unique identifier (e.g., "elem-4d3917a20a7405b")
  "element_type": "string",  // Type: title, narrative_text, list_item, etc.
  "text": "string",          // Element text content
  "metadata": {
    "page_number": number,
    "filename": "string",
    "coordinates": {         // Bounding box in PDF coordinate space
      "x0": number,          // Top-left X
      "y0": number,          // Top-left Y
      "x1": number,          // Bottom-right X
      "y1": number           // Bottom-right Y
    },
    "element_index": number,
    "additional": {
      "level": "string",     // Header level: "h1", "h2", etc.
      "font_size": "string"
    }
  }
}
```

## element_type Values
- `title` - Headings and section headers
- `narrative_text` - Body text paragraphs
- `list_item` - List items (bulleted/numbered)
- `page_break` - Page separators
- `image` - Images and figures
- `table` - Table data

## document.nodes[*]
```json
{
  "id": "string",
  "content": {
    "node_type": "string",
    "heading_level": number,
    "heading_text": "string"
  },
  "page": number,
  "bbox": {
    "x0": number,
    "y0": number,
    "x1": number,
    "y1": number
  }
}
```

## BBox Coordinates
- **PDF coordinate space**: (0,0) = bottom-left corner
- **Units**: Points (72pt = 1 inch)
- **Structure**: {x0, y0, x1, y1} = (left, top, right, bottom)

## Content Hierarchy
```
Full Text (content)
├── Page-Level (pages[])
│   ├── Page content
│   └── Page images
├── Element-Level (elements[])
│   ├── Semantic chunks
│   └── Spatial coordinates
└── Document-Level (metadata)
    ├── File metadata
    └── Page boundaries
```

## Common Tasks

### Get page 2 text
```python
b = doc['metadata']['pages']['boundaries'][1]
doc['content'][b['byte_start']:b['byte_end']]
```

### Get all titles
```python
[e['text'] for e in doc['elements'] if e['element_type'] == 'title']
```

### Get elements from specific page
```python
[e for e in doc['elements'] if e['metadata']['page_number'] == 1]
```

### Decode image (raw bytes)
```python
import numpy as np
img_data = np.array(doc['images'][0]['data'], dtype=np.uint8)
# Reshape based on width, height, and channels
```

### Extract all text from page
```python
page = next(p for p in doc['pages'] if p['page_number'] == 1)
page['content']
```
