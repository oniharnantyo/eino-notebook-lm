# JSON Structure Documentation

This document describes the structure of the `response.json` file output from PDF parsing.

---

## Root Level

The JSON file is an **array of document objects**, allowing multiple PDFs to be parsed and returned in a single response.

```json
[
  { /* Document Object */ }
]
```

---

## Document Object

### `content`
```json
"content": "<full extracted text from entire PDF>"
```
- **Type**: `string`
- **Purpose**: Raw text extraction from the entire PDF document
- **Use Case**: Full-text search, simple text analysis, LLM context
- **Note**: Contains all text from all pages concatenated

### `mime_type`
```json
"mime_type": "application/pdf"
```
- **Type**: `string`
- **Purpose**: Identifies the source document type
- **Common Values**: `application/pdf`, `image/png`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document`

### `quality_score`
```json
"quality_score": 1.0
```
- **Type**: `number` (0.0 to 1.0)
- **Purpose**: Confidence score of the parsing quality
- **Use Case**: Filtering low-quality results, deciding whether to re-parse

---

## `metadata` - Document Information

```json
"metadata": {
  "title": "string",
  "authors": ["string", ...],
  "created_by": "string",
  "format_type": "PDF 1.4",
  "pdf_version": "1.4",
  "producer": "string",
  "is_encrypted": boolean,
  "width": number,
  "height": number,
  "page_count": number,
  "output_format": "string",
  "quality_score": number,
  "pages": { /* see below */ }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Document title |
| `authors` | string[] | List of author names |
| `created_by` | string | Tool that created the PDF |
| `format_type` | string | PDF specification version |
| `pdf_version` | string | Alternative version field |
| `producer` | string | Software that produced PDF |
| `is_encrypted` | boolean | Security flag |
| `width` | number | Page width in points |
| `height` | number | Page height in points |
| `page_count` | number | Total number of pages |
| `output_format` | string | Response format type |
| `quality_score` | number | Duplicate of root-level score |

### `metadata.pages` - Byte-level Page Boundaries

```json
"pages": {
  "total_count": 26,
  "unit_type": "page",
  "boundaries": [
    {"byte_start": 4, "byte_end": 3487, "page_number": 1},
    {"byte_start": 3491, "byte_end": 7020, "page_number": 2}
  ],
  "pages": [
    {"number": 1, "dimensions": [612.0, 792.0], "is_blank": false}
  ]
}
```

| Sub-field | Type | Description |
|-----------|------|-------------|
| `total_count` | number | Total number of pages |
| `unit_type` | string | Unit of segmentation |
| `boundaries` | object[] | Byte offsets for each page in `content` |
| `pages` | object[] | Physical page properties |

**`boundaries` array**:
- Maps byte ranges in the root `content` field to page numbers
- Enables extracting text for specific pages without re-parsing
- **Example usage**:
  ```python
  # Extract page 2 text:
  page_2 = boundaries[1]
  page_2_text = content[page_2["byte_start"]:page_2["byte_end"]]
  ```

**`pages` array**:
- `number`: Page number (1-indexed)
- `dimensions`: `[width, height]` in points
- `is_blank`: Whether the page is empty

---

## `tables` - Structured Table Data

```json
"tables": null
```
- **Type**: `array` or `null`
- **Purpose**: Extracted tables with structure (rows, columns, headers)
- **Use Case**: Table data extraction, database insertion, analytics
- **Structure (when present)**:
  ```json
  [
    {
      "table_id": "string",
      "page_number": number,
      "bbox": {"x0": number, "y0": number, "x1": number, "y1": number},
      "headers": ["Column1", "Column2"],
      "rows": [
        ["Value1", "Value2"],
        ["Value3", "Value4"]
      ]
    }
  ]
  ```

---

## `images` - Embedded Images

```json
"images": [
  {
    "data": "<base64 encoded image bytes>",
    "format": "FlateDecode",
    "image_index": 0,
    "page_number": 1,
    "width": 10875,
    "height": 1275,
    "colorspace": "DeviceRGB",
    "bits_per_component": 8,
    "is_mask": false
  }
]
```

| Field | Type | Description |
|-------|------|-------------|
| `data` | string | Base64-encoded image bytes |
| `format` | string | Compression format (FlateDecode, DCTDecode, etc.) |
| `image_index` | number | Image sequence number in document |
| `page_number` | number | Which page the image appears on |
| `width` | number | Image width in pixels |
| `height` | number | Image height in pixels |
| `colorspace` | string | Color model (DeviceRGB, DeviceGray, etc.) |
| `bits_per_component` | number | Color depth (usually 8) |
| `is_mask` | boolean | Whether it's a transparency mask |

---

## `pages` - Page-Level Detail

```json
"pages": [
  {
    "page_number": 1,
    "content": "<text content for this page only>",
    "images": [],
    "is_blank": false,
    "hierarchy": {
      "block_count": 2,
      "blocks": [
        {
          "text": "AgenticOCR...",
          "font_size": "12.0",
          "level": "h1",
          "bbox": {
            "x0": 100, "y0": 50,
            "x1": 500, "y1": 80
          }
        }
      ]
    }
  }
]
```

| Field | Type | Description |
|-------|------|-------------|
| `page_number` | number | 1-indexed page number |
| `content` | string | Text content for this page only |
| `images` | array | Images appearing on this page |
| `is_blank` | boolean | Whether the page is empty |
| `hierarchy` | object | Layout structure with text blocks |

### `hierarchy` Object

| Sub-field | Type | Description |
|-----------|------|-------------|
| `block_count` | number | Number of text blocks |
| `blocks` | object[] | Array of text block objects |

**Block object structure**:
- `text`: The text content
- `font_size`: Font size in points
- `level`: Heading level (h1, h2, etc.) or null
- `bbox`: Bounding box coordinates

---

## `elements` - Semantic Segmentation

```json
"elements": [
  {
    "element_id": "elem-4d3917a20a7405b",
    "element_type": "title",
    "text": "1",
    "metadata": {
      "page_number": 1,
      "filename": "AgenticOCR...",
      "coordinates": {
        "x0": 303.26, "y0": 40.12,
        "x1": 308.24, "y1": 55.15
      },
      "element_index": 0,
      "additional": {
        "level": "h2",
        "font_size": "3.9875333"
      }
    }
  }
]
```

### Element Types

| Type | Description |
|------|-------------|
| `title` | Headings and titles |
| `narrative_text` | Body paragraphs |
| `list_item` | Bulleted or numbered list items |
| `page_break` | Page separators |
| `image` | Image references |

### Element Fields

| Field | Type | Description |
|-------|------|-------------|
| `element_id` | string | Unique identifier |
| `element_type` | string | Semantic category (see table above) |
| `text` | string | Text content of the element |
| `metadata` | object | Additional metadata (see below) |

### `metadata` Object (within element)

| Field | Type | Description |
|-------|------|-------------|
| `page_number` | number | Page containing this element |
| `filename` | string | Source document filename |
| `coordinates` | object | Position on page (bbox) |
| `element_index` | number | Sequence in document |
| `additional` | object | Extra properties |

**`coordinates` object**: Bounding box in PDF coordinate space
- `x0`: Left edge
- `y0`: Bottom edge
- `x1`: Right edge
- `y1`: Top edge

**`additional` object**: Type-specific properties
- `level`: Heading level (for titles)
- `font_size`: Font size in points

---

## `document` - Node Graph Structure

```json
"document": {
  "nodes": [
    {
      "id": "node-4ea73a8362baeb8e",
      "content": {
        "node_type": "group",
        "heading_level": 2,
        "heading_text": "1"
      },
      "page": 1,
      "bbox": {
        "x0": 303.26, "y0": 40.12,
        "x1": 308.24, "y1": 55.15
      }
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `nodes` | array | Graph nodes representing document structure |

### Node Object

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique node identifier |
| `content` | object | Node content (type-specific) |
| `page` | number | Page number |
| `bbox` | object | Bounding box coordinates |

---

## Bounding Box (bbox) Coordinate System

All spatial data uses PDF coordinate space:

```
(0, height) ──────────────── (width, height)
     │                           │
     │      PDF Page             │
     │                           │
     │                           │
(0, 0) ──────────────────── (width, 0)
```

- **Origin**: Bottom-left corner
- **Units**: Points (1 inch = 72 points)
- **Structure**:
  ```json
  {
    "x0": number,  // Left edge
    "y0": number,  // Bottom edge
    "x1": number,  // Right edge
    "y1": number   // Top edge
  }
  ```

---

## Usage Examples

### Extract page text using boundaries
```python
import json

with open('response.json') as f:
    data = json.load(f)

doc = data[0]
boundaries = doc['metadata']['pages']['boundaries']
content = doc['content']

# Get page 2 text
page_2 = boundaries[1]
page_2_text = content[page_2['byte_start']:page_2['byte_end']]
```

### Extract all titles
```python
titles = [
    elem['text']
    for elem in doc['elements']
    if elem['element_type'] == 'title'
]
```

### Decode and save image
```python
import base64

for img in doc['images']:
    img_data = base64.b64decode(img['data'])
    with open(f"image_{img['image_index']}.png", 'wb') as f:
        f.write(img_data)
```

---

## Data Flow Diagram

```
PDF File
    │
    ├─→ content (full text)
    │
    ├─→ metadata.pages.boundaries (byte offsets)
    │       │
    │       └─→ Extract page text from content
    │
    ├─→ pages[] (per-page content + hierarchy)
    │
    ├─→ elements[] (semantic chunks)
    │       ├─→ titles
    │       ├─→ narrative_text
    │       ├─→ list_item
    │       └─→ page_break
    │
    ├─→ images[] (base64 encoded)
    │
    └─→ document.nodes[] (graph structure)
```

---

## Insights

### Redundant by Design
The same content exists in multiple places, each optimized for different access patterns:

| Field | Access Pattern |
|-------|----------------|
| `content` | Full-text search, simple analysis |
| `pages[].content` | Page-aware processing |
| `elements[].text` | Semantic chunking for RAG |

### Element Metadata
The `elements[].metadata.coordinates` enables visual reconstruction - you can draw colored boxes over the original PDF showing where each element was found.

### Why Multiple Representations?
- **`content`**: Simple, fast text access
- **`pages`**: Layout-aware processing
- **`elements`**: Semantic understanding (titles vs paragraphs vs lists)
- **`document.nodes`**: Complex relationships and document structure

---

# RAG Improvement Guide

This section explains how each part of the JSON structure can be used to improve Retrieval-Augmented Generation (RAG) systems.

## Priority Matrix

| Component | RAG Value | Effort | Priority |
|-----------|-----------|--------|----------|
| `elements[]` | ⭐⭐⭐ | Low | 1st |
| `metadata.pages.boundaries` | ⭐⭐ | Low | 2nd |
| `images[]` | ⭐⭐⭐ | Medium | 3rd |
| `tables[]` | ⭐⭐⭐ | Medium | 4th |
| `pages[].hierarchy` | ⭐⭐ | Medium | 5th |
| `document.nodes[]` | ⭐ | High | 6th |

---

## 1. `elements[]` - Semantic Chunking ⭐⭐⭐

### Why It's Powerful
Traditional RAG splits text by token count, breaking semantic boundaries. Using `elements[]` preserves natural document structure.

### RAG Benefits

| Use Case | Implementation |
|----------|----------------|
| **Semantic Chunking** | Use `element_type` to create meaningful chunks |
| **Context Preservation** | Keep related elements under same `title` together |
| **Metadata Filtering** | Filter by type (e.g., exclude `page_break`, search only `narrative_text`) |
| **Source Citation** | Return `page_number` and `coordinates` with answers |
| **Chunk Enrichment** | Add `element_type` as metadata to embeddings |

### Element Type Strategies

```python
# Create RAG chunks by element type
chunks = []
for elem in doc['elements']:
    if elem['element_type'] in ['narrative_text', 'title', 'list_item']:
        chunks.append({
            'id': elem['element_id'],
            'text': elem['text'],
            'type': elem['element_type'],
            'page': elem['metadata']['page_number'],
            'metadata': {
                'element_type': elem['element_type'],
                'page': elem['metadata']['page_number'],
                'coordinates': elem['metadata']['coordinates']
            }
        })
```

### Search Scopes by Element Type

| Query Type | Search In | Reason |
|------------|-----------|--------|
| Content questions | `narrative_text` + `list_item` | Main content |
| Outline/Navigation | `title` | Document structure |
| Definitions | `narrative_text` near `title` | Context-aware |
| Visual content | `image` elements | Figures/charts |

---

## 2. `metadata.pages.boundaries` - Dynamic Context ⭐⭐

### Why It's Powerful
When a chunk is retrieved, you can expand to full page context for better generation.

### RAG Benefits

| Use Case | Implementation |
|----------|----------------|
| **Full Page Context** | Expand retrieved chunk to full page |
| **Page-Level Retrieval** | Retrieve full pages for overview queries |
| **Context Expansion** | Add neighboring pages for complex answers |

### Implementation

```python
def get_page_context(doc, page_number):
    """Get full page text for context expansion"""
    boundaries = doc['metadata']['pages']['boundaries']
    page_boundary = boundaries[page_number - 1]  # 1-indexed
    content = doc['content']
    return content[page_boundary['byte_start']:page_boundary['byte_end']]

# After retrieval, expand context
def expand_context(retrieved_chunk, doc):
    page_text = get_page_context(doc, retrieved_chunk['page'])
    return {
        'chunk': retrieved_chunk['text'],
        'full_page': page_text,
        'source': f"Page {retrieved_chunk['page']}"
    }
```

---

## 3. `images[]` - Multimodal RAG ⭐⭐⭐

### Why It's Powerful
Documents contain visual information (charts, diagrams, figures) that text alone misses.

### RAG Benefits

| Use Case | Implementation |
|----------|----------------|
| **Figure Extraction** | Include charts/figures in retrieval |
| **Image Embeddings** | Use CLIP/VLM for image-text queries |
| **Cross-Modal Retrieval** | Text query → retrieve relevant images |
| **Visual Context** | Provide images to LLM alongside text |

### Implementation

```python
import base64

# Create image chunks for multimodal RAG
image_chunks = []
for img in doc['images']:
    image_chunks.append({
        'id': f"img-{img['image_index']}",
        'page': img['page_number'],
        'data': img['data'],  # base64
        'format': img['format'],
        'dimensions': (img['width'], img['height'])
    })

# For VLM input
def prepare_multimodal_context(text_chunks, image_chunks):
    context = {
        'text': [c['text'] for c in text_chunks],
        'images': [base64.b64decode(i['data']) for i in image_chunks]
    }
    return context
```

---

## 4. `tables[]` - Structured Data RAG ⭐⭐⭐

### Why It's Powerful
Tables contain structured data (numerical, categorical) that's often mangled in plain text extraction.

### RAG Benefits

| Use Case | Implementation |
|----------|----------------|
| **Table QA** | Direct table lookup for numerical queries |
| **Data Extraction** | Convert to structured format for SQL |
| **Table Summarization** | Pass structured table to LLM |

### Implementation

```python
def process_table(table):
    """Convert table to queryable format"""
    return {
        'id': table['table_id'],
        'page': table['page_number'],
        'headers': table['headers'],
        'rows': table['rows'],
        # Create lookup dictionary
        'lookup': {
            row[i]: row
            for row in table['rows']
            for i in range(len(table['headers']))
        }
    }

# Query example
def query_table(table, column, value):
    """Find row where column matches value"""
    col_idx = table['headers'].index(column)
    for row in table['rows']:
        if row[col_idx] == value:
            return row
    return None
```

---

## 5. `pages[].hierarchy.blocks` - Layout-Aware Chunking ⭐⭐

### Why It's Powerful
Font size and level indicate importance - headings should be weighted higher in retrieval.

### RAG Benefits

| Use Case | Implementation |
|----------|----------------|
| **Importance Weighting** | Boost chunks with larger `font_size` |
| **Heading Hierarchy** | Build document tree for hierarchical retrieval |
| **Spatial Grouping** | Group nearby blocks (by `bbox`) together |

### Implementation

```python
def build_document_tree(pages):
    """Build hierarchical document structure"""
    tree = {}
    for page in pages:
        for block in page['hierarchy']['blocks']:
            level = block.get('level')
            if level:  # It's a heading
                if level not in tree:
                    tree[level] = []
                tree[level].append({
                    'text': block['text'],
                    'page': page['page_number'],
                    'bbox': block['bbox']
                })
    return tree

# Weight chunks by font size
def chunk_weight(chunk):
    font_size = float(chunk.get('font_size', 12))
    return 1 + (font_size - 12) / 12  # Boost larger fonts
```

---

## 6. `document.nodes[]` - Graph-Based RAG ⭐

### Why It's Powerful
Documents have relationships (sections, subsections) that graph traversal can exploit.

### RAG Benefits

| Use Case | Implementation |
|----------|----------------|
| **Section-Aware Retrieval** | Retrieve all content under a heading |
| **Document Traversal** | Follow related nodes for expanded context |
| **Graph RAG** | Build knowledge graph from structure |

### Implementation

```python
def build_section_graph(nodes):
    """Build graph of document sections"""
    graph = {}
    for node in nodes:
        node_id = node['id']
        content = node['content']
        graph[node_id] = {
            'type': content.get('node_type'),
            'heading_level': content.get('heading_level'),
            'page': node['page'],
            'children': []
        }
    return graph

def get_section_content(graph, heading_node_id):
    """Get all content under a heading"""
    node = graph[heading_node_id]
    section = {'heading': node, 'content': []}
    # Recursively get children
    for child_id in node['children']:
        section['content'].append(get_section_content(graph, child_id))
    return section
```

---

## Enhanced RAG Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│                      QUERY ANALYSIS                              │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌─────────────────────┼─────────────────────┐
          │                     │                     │
          ▼                     ▼                     ▼
    ┌───────────┐         ┌───────────┐         ┌───────────┐
    │ Textual   │         │ Numerical │         │ Visual    │
    │ Query     │         │ Query     │         │ Query     │
    └─────┬─────┘         └─────┬─────┘         └─────┬─────┘
          │                     │                     │
          ▼                     ▼                     ▼
    ┌───────────┐         ┌───────────┐         ┌───────────┐
    │ Search    │         │ Search    │         │ Search    │
    │ elements  │         │ tables    │         │ images    │
    │ + pages   │         │           │         │           │
    └─────┬─────┘         └─────┬─────┘         └─────┬─────┘
          │                     │                     │
          └─────────────────────┼─────────────────────┘
                                ▼
                    ┌───────────────────────┐
                    │  Combine Results      │
                    │  with Metadata        │
                    │  - page_number        │
                    │  - element_type       │
                    │  - coordinates        │
                    └───────────┬───────────┘
                                ▼
                    ┌───────────────────────┐
                    │  Expand Context       │
                    │  (full page,          │
                    │   neighboring pages)  │
                    └───────────┬───────────┘
                                ▼
                    ┌───────────────────────┐
                    │  LLM Generate         │
                    │  + Citations          │
                    └───────────────────────┘
```

---

## Complete Implementation Example

```python
import json
from typing import List, Dict, Any

class EnhancedRAG:
    def __init__(self, parsed_json_path: str):
        with open(parsed_json_path) as f:
            data = json.load(f)
        self.doc = data[0]
        self._build_indexes()

    def _build_indexes(self):
        """Build efficient lookup indexes"""
        # Element index by type
        self.elements_by_type = {}
        for elem in self.doc['elements']:
            etype = elem['element_type']
            if etype not in self.elements_by_type:
                self.elements_by_type[etype] = []
            self.elements_by_type[etype].append(elem)

        # Page boundary index
        self.page_boundaries = {
            b['page_number']: b
            for b in self.doc['metadata']['pages']['boundaries']
        }

    def get_chunks(self, types: List[str] = None) -> List[Dict]:
        """Get chunks by element type"""
        if types is None:
            types = ['narrative_text', 'title', 'list_item']

        chunks = []
        for elem in self.doc['elements']:
            if elem['element_type'] in types:
                chunks.append({
                    'id': elem['element_id'],
                    'text': elem['text'],
                    'type': elem['element_type'],
                    'page': elem['metadata']['page_number'],
                    'metadata': {
                        'element_type': elem['element_type'],
                        'page': elem['metadata']['page_number'],
                        'coordinates': elem['metadata']['coordinates']
                    }
                })
        return chunks

    def get_page_context(self, page_number: int) -> str:
        """Get full page text for context"""
        boundary = self.page_boundaries[page_number]
        return self.doc['content'][
            boundary['byte_start']:boundary['byte_end']
        ]

    def expand_with_context(self, chunk: Dict, window: int = 0) -> Dict:
        """Expand chunk with page context"""
        page_num = chunk['page']
        context = self.get_page_context(page_num)
        return {
            **chunk,
            'page_context': context,
            'source': f"Page {page_num}"
        }

    def get_tables(self) -> List[Dict]:
        """Get all tables for structured queries"""
        return self.doc.get('tables', [])

    def get_images(self) -> List[Dict]:
        """Get all images for multimodal RAG"""
        return self.doc.get('images', [])

# Usage
rag = EnhancedRAG('response.json')

# Get semantic chunks
chunks = rag.get_chunks()

# Expand with context
enriched = [rag.expand_with_context(c) for c in chunks[:5]]

# Query for numerical data in tables
tables = rag.get_tables()

# Get images for visual queries
images = rag.get_images()
```

---

## Best Practices

### 1. Metadata Enrichment
Add element metadata to your vector store:
```python
# When embedding
vector_store.upsert([
    {
        'id': chunk['id'],
        'vector': embedding,
        'metadata': {
            'text': chunk['text'],
            'element_type': chunk['type'],
            'page': chunk['page']
        }
    }
])
```

### 2. Hybrid Search
Combine semantic search with metadata filters:
```python
# Search only in narrative text
results = vector_store.search(
    query_embedding,
    filter={'element_type': 'narrative_text'}
)
```

### 3. Citation Generation
Use coordinates for precise citations:
```python
citation = f"Page {elem['metadata']['page_number']}, " \
           f"coords: {elem['metadata']['coordinates']}"
```

### 4. Context Expansion Strategy
- Start with element chunk
- If low confidence, expand to full page
- If still low, include neighboring pages

---

## Key Insights

1. **Element-Type Filtering**: Implement search scopes by filtering `elements[]` - search in `narrative_text` for content, `title` for navigation

2. **Page Context as Fallback**: Use `elements[]` for granular retrieval, expand to full `pages[]` when LLM needs more context

3. **Metadata as Embedding Boost**: Add `element_type`, `page_number`, `font_size` as vector store metadata for filtering/boosting at query time

4. **Multimodal Integration**: Combine text from `elements[]` with images from `images[]` for visual documents

5. **Structured Data Separation**: Handle `tables[]` separately with SQL/query engines for numerical accuracy

---

## Priority Implementation Order

A phased approach to implementing RAG improvements using this JSON structure.

### Phase 1: Foundation (Week 1) ⭐⭐⭐
**High Impact, Low Effort**

| Task | Component | Effort | Impact |
|------|-----------|--------|--------|
| 1.1 | Semantic chunking with `elements[]` | 2 days | High |
| 1.2 | Add metadata to embeddings (`element_type`, `page`) | 1 day | High |
| 1.3 | Implement element-type filtering | 1 day | Medium |

**Deliverables**:
- Basic RAG with semantic chunks
- Filter by `narrative_text`, `title`, `list_item`
- Citations with page numbers

**Code Example**:
```python
# Phase 1: Basic semantic chunking
def phase1_chunks(doc):
    return [{
        'id': e['element_id'],
        'text': e['text'],
        'metadata': {
            'type': e['element_type'],
            'page': e['metadata']['page_number']
        }
    } for e in doc['elements']
    if e['element_type'] in ['narrative_text', 'title', 'list_item']]
```

---

### Phase 2: Context Expansion (Week 2) ⭐⭐
**Medium Impact, Low Effort**

| Task | Component | Effort | Impact |
|------|-----------|--------|--------|
| 2.1 | Implement page context expansion | 1 day | High |
| 2.2 | Add boundary-based page lookup | 1 day | Medium |
| 2.3 | Implement confidence-based expansion | 2 days | Medium |

**Deliverables**:
- Expand chunks to full page context
- Dynamic context window based on query complexity
- Improved answer quality

**Code Example**:
```python
# Phase 2: Context expansion
def phase2_expand(chunk, doc, confidence_threshold=0.7):
    if chunk['confidence'] < confidence_threshold:
        # Expand to full page
        page_num = chunk['metadata']['page']
        boundary = doc['metadata']['pages']['boundaries'][page_num - 1]
        page_text = doc['content'][boundary['byte_start']:boundary['byte_end']]
        chunk['expanded_context'] = page_text
    return chunk
```

---

### Phase 3: Multimodal Support (Week 3-4) ⭐⭐⭐
**High Impact, Medium Effort**

| Task | Component | Effort | Impact |
|------|-----------|--------|--------|
| 3.1 | Extract and index images | 2 days | High |
| 3.2 | Implement image embeddings (CLIP) | 3 days | High |
| 3.3 | Cross-modal retrieval (text→image) | 3 days | Medium |
| 3.4 | Multimodal LLM integration | 2 days | High |

**Deliverables**:
- Image search capability
- Figure/chart extraction for visual documents
- Multimodal context for LLM

**Code Example**:
```python
# Phase 3: Multimodal support
import base64
from PIL import Image
import clip

def phase3_multimodal(doc, query):
    # Text chunks
    text_chunks = phase1_chunks(doc)

    # Image chunks
    image_chunks = []
    for img in doc['images']:
        image_data = base64.b64decode(img['data'])
        image_chunks.append({
            'id': f"img-{img['image_index']}",
            'page': img['page_number'],
            'image': Image.open(io.BytesIO(image_data))
        })

    # Cross-modal search
    text_results = search_text(text_chunks, query)
    image_results = search_images(image_chunks, query)

    return text_results + image_results
```

---

### Phase 4: Structured Data (Week 5) ⭐⭐⭐
**High Impact, Medium Effort**

| Task | Component | Effort | Impact |
|------|-----------|--------|--------|
| 4.1 | Parse and normalize tables | 2 days | High |
| 4.2 | Implement table-specific QA | 2 days | High |
| 4.3 | SQL/query interface for tables | 3 days | Medium |

**Deliverables**:
- Table extraction and indexing
- Numerical query support
- Structured data output

**Code Example**:
```python
# Phase 4: Table handling
import pandas as pd

def phase4_tables(doc):
    tables = []
    for table in doc.get('tables', []):
        df = pd.DataFrame(
            table['rows'],
            columns=table['headers']
        )
        tables.append({
            'id': table['table_id'],
            'page': table['page_number'],
            'dataframe': df
        })
    return tables

def query_table(tables, query):
    """Natural language to table query"""
    # Detect numerical queries
    if is_numerical_query(query):
        # Route to table search
        for table in tables:
            result = table_search(table['dataframe'], query)
            if result:
                return result
    return None
```

---

### Phase 5: Advanced Features (Week 6+) ⭐
**Medium Impact, High Effort**

| Task | Component | Effort | Impact |
|------|-----------|--------|--------|
| 5.1 | Build document hierarchy tree | 3 days | Medium |
| 5.2 | Hierarchical retrieval | 4 days | Medium |
| 5.3 | Graph-based RAG with `document.nodes[]` | 5 days | Low-Medium |
| 5.4 | Layout-aware chunking with `hierarchy.blocks` | 3 days | Low |

**Deliverables**:
- Section-aware retrieval
- Document structure navigation
- Font-size based weighting

**Code Example**:
```python
# Phase 5: Advanced features
def phase5_hierarchy(doc):
    """Build document hierarchy from elements"""
    tree = {'root': {'children': [], 'content': []}}
    current_path = ['root']

    for elem in doc['elements']:
        if elem['element_type'] == 'title':
            level = elem['metadata']['additional'].get('level', 'h3')
            # Navigate to correct level
            while len(current_path) > 1:
                parent = get_node(tree, current_path[-1])
                if parent.get('level') <= level:
                    break
                current_path.pop()

            # Create new section
            section = {
                'id': elem['element_id'],
                'level': level,
                'title': elem['text'],
                'page': elem['metadata']['page_number'],
                'children': [],
                'content': []
            }
            parent = get_node(tree, current_path[-1])
            parent['children'].append(section)
            current_path.append(section['id'])
        else:
            # Add content to current section
            node = get_node(tree, current_path[-1])
            node['content'].append(elem)

    return tree
```

---

### Implementation Timeline Summary

```
Week 1-2:  ████████████░░░░░░░░░░░░░░  Foundation + Context
Week 3-4:  ░░░░░░░░░░░░████████████░░░  Multimodal Support
Week 5:    ░░░░░░░░░░░░░░░░░░░░░░░░░░░  Structured Data
Week 6+:   ░░░░░░░░░░░░░░░░░░░░░░░░░░░  Advanced Features
```

### Milestone Checklist

| Phase | Milestone | Success Criteria |
|-------|-----------|------------------|
| 1 | Basic RAG | Can retrieve and answer with citations |
| 2 | Context Expansion | Answers improve with page context |
| 3 | Multimodal | Can find and use images in answers |
| 4 | Tables | Numerical queries return accurate values |
| 5 | Advanced | Hierarchical navigation working |

### Resource Requirements

| Phase | Skills Needed | Tools |
|-------|---------------|-------|
| 1 | Python, Vector DB | Chroma/Pinecone/Weaviate |
| 2 | Python | Same as Phase 1 |
| 3 | CLIP/VLM, Image processing | OpenAI CLIP, GPT-4V |
| 4 | Pandas, SQL | Pandas, SQLite/Postgres |
| 5 | Graph algorithms | NetworkX, LangChain |

### Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Low-quality parsing | High | Check `quality_score`, re-parse if < 0.8 |
| Large images | Medium | Resize before embedding |
| Complex tables | Medium | Fallback to LLM extraction |
| Missing metadata | Low | Use defaults (page=1, type=unknown) |

### Testing Strategy

```python
# Test each phase
def test_phase(phase_number, doc, test_queries):
    print(f"Testing Phase {phase_number}")

    for query in test_queries:
        result = execute_query(doc, query, phase=phase_number)

        # Evaluate
        relevance = evaluate_relevance(query, result)
        citation = has_citation(result)
        confidence = result.get('confidence', 0)

        print(f"Query: {query}")
        print(f"  Relevance: {relevance:.2f}")
        print(f"  Citation: {citation}")
        print(f"  Confidence: {confidence:.2f}")
        print()

# Test queries by phase
test_sets = {
    1: ["What is the main topic?"],
    2: ["Explain section 2 in detail"],
    3: ["What does Figure 1 show?"],
    4: ["What is the revenue in 2023?"],
    5: ["Summarize all subsections of chapter 3"]
}
```

---

## Quick Start: Minimal Viable RAG

If you need to implement RAG quickly, start with this minimal setup:

```python
# Minimal RAG using elements[] (1 day implementation)
import json

def minimal_rag(json_path, query, top_k=3):
    with open(json_path) as f:
        doc = json.load(f)[0]

    # Get text chunks from elements
    chunks = [
        e['text']
        for e in doc['elements']
        if e['element_type'] in ['narrative_text', 'title', 'list_item']
    ]

    # Simple keyword search (replace with embeddings)
    scores = [
        len(set(query.lower().split()) & set(chunk.lower().split()))
        for chunk in chunks
    ]

    # Get top results
    top_indices = sorted(range(len(scores)), key=lambda i: -scores[i])[:top_k]
    results = [
        {
            'text': chunks[i],
            'score': scores[i]
        }
        for i in top_indices
    ]

    return results

# Usage
results = minimal_rag('response.json', 'machine learning')
```

**Upgrade Path**: Replace the keyword search with proper embeddings, then add features from each phase.
