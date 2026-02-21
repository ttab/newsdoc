# NewsDoc

This package provides type declarations for NewsDoc as [Go types](doc.go), [protobuf](newsdoc.proto) messages, and a [JSON schema](newsdoc.schema.json). Protobuf and JSON schemas are generated from the Go type declarations.

NewsDoc was created to be a convenient and type-safe document format for editorial data like articles and concept metadata that minimises the need for evolving the schema to adapt to new types of data. It avoids this by not using data structure for expressing relationships (`{categories:['a', 'b'], seeAlso:['c', 'd']}`) or type/identity of the data (`{articleMetadata:{teaserHeadline:"v", teaserText:"w"}, headline:"x", "lead_in":"y", paragraphs:["z"]}`). An example of a hypothetical format that does this:

``` json
{
    "categories": [
        "28b94216-77d7-41e9-be08-a6bfbe59f1d5",
        "a23528b7-31af-4ae2-bbca-0c78f1cbc959",
    ],
    "readMore": [
        "6dd826dd-d866-459b-a07e-0da4bad7bce0",
        "043c248f-92ac-4e0b-b0ec-76cc26323634"
    ],
    "articleMetadata": {
        "teaserHeadline": "v",
        "teaserText": "w"
    },
    "headline": "x",
    "lead_in": "y",
    "paragraphs": ["z"],
    "image": "https://example.com/an-image.jpg",
    "image_width": 128,
    "image_height": 128,
    "image_alt_text": "desc"
}
```

Instead it adopts a view of documents as a set of links expressing relationships to other entities, a set of typed metadata blocks, and a list of typed content blocks that represent the actual content of f.ex. an article. The article hinted at in the above paragraph would instead look like this:

``` json
{
    "type": "example/article",
    "links": [
        {"rel":"category", "uuid":"28b94216-77d7-41e9-be08-a6bfbe59f1d5"},
        {"rel":"category", "uuid":"a23528b7-31af-4ae2-bbca-0c78f1cbc959"},
        {
            "rel":"see-also", "type":"example/article",
            "uuid":"6dd826dd-d866-459b-a07e-0da4bad7bce0"
        },
        {
            "rel":"see-also", "type":"example/article",
            "uuid":"043c248f-92ac-4e0b-b0ec-76cc26323634"
        }
    ],
    "meta": [
        {
            "type": "example/teaser",
            "title": "v",
            "data": {
                "text": "w"
            }
        }
    ],
    "content": [
        {
            "type": "example/headline",
            "data": {
                "text": "x"
            }
        },
        {
            "type": "example/image",
            "url": "https://example.com/an-image.jpg",
            "data": {
                "width": "128",
                "height": "128",
                "alt": "desc"
            }
        },
        {
            "type": "example/lead-in",
            "data": {
                "text": "y"
            }
        },
        {
            "type": "example/paragraph",
            "data": {
                "text": "z"
            }
        },
    ]
}
```

This kind of structure allows a system that's using NewsDoc to f.ex. recognise that there is a link to another entity, or a content element with text, without knowing about the specific type of relationship or content. On the flip side it's also easy to ignore f.ex. a metadata block with a type that you don't recognize.

One thing is lost in translation here, the "data" object of a block is a string->string key value structure, so the width `128` becomes `"128"`. We sacrifice the specific types of some data to be able to have a largely static type system. But the "type contract" between content producers and consumers in a system like this is that "width" and "height" always must be integers. [Revisor](https://github.com/ttab/revisor) is our attempt to formalise and enforce these type contracts.

A revisor schema for the above format could look like this:

``` json
{"documents":[{
  "name": "News article",
  "description": "A basic news article example",
  "declares": "example/article",
  "links": [
    {
      "name": "Category",
      "description": "A category assigned to the article",
      "declares": {"rel":"category"},
      "attributes": {"uuid": {}}
    }
    {
      "name": "Read more",
      "description": "A link to other articles that are interesting",
      "declares": {"rel":"see-also", "type": "example/article"},
      "attributes": {"uuid": {}}
    }
  ],
  "meta": [
    {
      "name": "Teaser",
      "declares": {"type":"example/teaser"},
      "attributes": {"title": {}},
      "data": {"text": {}},
      "count": 1
    }
  ],
  "content": [
    {
      "name": "Headline",
      "declares": {"type":"example/headline"},
      "data": {"text": {}}
    },
    {
      "name": "Lead-in",
      "declares": {"type":"example/lead-in"},
      "data": {"text": {}}
    },
    {
      "name": "Paragraph",
      "declares": {"type":"example/paragraph"},
      "data": {"text": {}}
    },
    {
      "name": "Image",
      "declares": {"type":"example/image"},
      "attributes": {
        "url": {"glob":"https://**"}
      },
      "data": {
        "width": {"format":"int"},
        "height": {"format":"int"},
        "alt": {},
      }
    }
  ]
}]}
```

This schema can then be used to validate documents to ensure the data quality of stored documents. It's also serves as documentation, and can be used by automated systems like a full text index provide a hint about the correct way to index the data.

## Value extractor expressions

The `ValueExtractor` provides a way to extract values from documents using a selector expression language. An expression consists of a chain of block selectors followed by a value specifier that determines what to extract from the matched blocks.

### Selectors

Selectors navigate the block hierarchy of a document. Each selector targets a block list (`meta`, `links`, or `content`) and can optionally filter by block attributes:

```
.meta                              -- all meta blocks
.links(rel='category')             -- links with rel "category"
.meta(type='core/note').links      -- links inside meta blocks of type "core/note"
.content(type='core/text' role='heading')  -- content blocks matching both type and role
```

Selectors can be chained to navigate into nested blocks. The available filter attributes are: `id`, `uuid`, `uri`, `url`, `type`, `rel`, `role`, `name`, `value`, `contenttype`, and `sensitivity`.

### Extracting data values

Use `.data{}` to extract values from the matched blocks' data maps:

```
.meta(type='core/planning-item').data{start_date, end_date}
```

Each matched block must have all specified data keys for the extraction to succeed. Append `?` to make a value optional:

```
.meta(type='core/planning-item').data{start_date, date_tz?}
```

### Extracting block attributes

Use `@{}` to extract block attribute values:

```
.content(type='core/text')@{value}
.links(rel='author')@{uuid, title}
```

When no selectors are provided, `@{}` extracts document-level attributes (`uuid`, `type`, `uri`, `url`, `title`, `language`):

```
@{title, language}
```

### Annotations and roles

Values can be annotated with a type hint using `:`, and given a role using `=` as a prefix:

```
.meta(type='core/event').data{date:date, tz=date_timezone?}
```

Here `date` has the annotation `date`, and `date_timezone` is extracted with the role `tz`. Annotations and roles are passed through in the extracted results and can be used by the caller to interpret the values.

### Extracting full blocks

If no `.data{}` or `@{}` value specifier is present, the expression extracts the full matched blocks. Block extraction requires a name prefix and optionally accepts an annotation:

```
name=.selectors
name=.selectors:annotation
```

Examples:

```
items=.meta(type='core/collection').links(rel='item')
event=.links(rel='event' type='core/event'):calendar
```

The name is used as the key in the extracted results and populates the `Name` field of the `ExtractedValue`. The matched block is available in the `Block` field.
