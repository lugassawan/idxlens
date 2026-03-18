# Batch Processing

IDXLens has a built-in `batch` command for processing multiple PDFs with bounded concurrency.

## Using the batch command

```sh
# Process all PDFs in a directory
idxlens batch "reports/*.pdf" --output-dir results/

# Use 8 workers for faster processing
idxlens batch "reports/*.pdf" --workers 8 --output-dir results/

# Specify report type and output format
idxlens batch "reports/*.pdf" --type balance-sheet --format csv --output-dir results/
```

The `batch` command outputs a JSON summary showing the count of successful and failed files.

## Shell scripting alternatives

For more control over the processing workflow, you can use shell scripting.

## Process all PDFs in a directory

```sh
for pdf in reports/*.pdf; do
    echo "Processing: $pdf"
    idxlens extract financial "$pdf" --output "${pdf%.pdf}.json"
done
```

This creates a `.json` file alongside each `.pdf` file.

## Process with CSV output

```sh
for pdf in reports/*.pdf; do
    idxlens extract financial "$pdf" --format csv --output "${pdf%.pdf}.csv"
done
```

## Classify all reports first

Before extracting, classify all PDFs to understand what you have:

```sh
for pdf in reports/*.pdf; do
    type=$(idxlens classify "$pdf" --format json | jq -r '.type')
    confidence=$(idxlens classify "$pdf" --format json | jq -r '.confidence')
    echo "$pdf: $type ($confidence)"
done
```

## Process only specific report types

Extract only balance sheets from a directory of mixed reports:

```sh
for pdf in reports/*.pdf; do
    type=$(idxlens classify "$pdf" --format json | jq -r '.type')
    if [ "$type" = "balance-sheet" ]; then
        idxlens extract financial "$pdf" --type balance-sheet --output "${pdf%.pdf}.json"
    fi
done
```

## Parallel processing

Use `xargs` to process multiple files in parallel:

```sh
find reports/ -name "*.pdf" | xargs -P 4 -I {} sh -c '
    idxlens extract financial "{}" --output "{}.json"
'
```

The `-P 4` flag runs up to 4 processes in parallel.

## Merge results

Combine multiple JSON outputs into a single array:

```sh
jq -s '.' reports/*.json > combined.json
```

## Error handling

Skip files that fail and log errors:

```sh
for pdf in reports/*.pdf; do
    if ! idxlens extract financial "$pdf" --output "${pdf%.pdf}.json" 2>>"errors.log"; then
        echo "FAILED: $pdf" | tee -a errors.log
    fi
done
```

## Summary report

Generate a summary of all processed files:

```sh
for json in reports/*.json; do
    company=$(jq -r '.company // "unknown"' "$json")
    type=$(jq -r '.type' "$json")
    items=$(jq '.items | length' "$json")
    echo "$json: $company ($type, $items items)"
done
```
