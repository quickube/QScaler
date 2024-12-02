#!/bin/bash

# Define source and target file pairs as arrays
SOURCE_FILES=(
  "./config/crd/bases/quickube.com_qworkers.yaml"
  "./config/crd/bases/quickube.com_scalerconfigs.yaml"
)
TARGET_FILES=(
  "./helm/templates/crds/qworkers.yaml"
  "./helm/templates/crds/scalerconfigs.yaml"
)

# Function to replace target file content
replace_file_content() {
  local source_file="$1"
  local target_file="$2"

  # Check if source file exists
  if [[ ! -f "$source_file" ]]; then
    echo "Source file $source_file does not exist. Skipping."
    return
  fi

  # Check if target file exists
  if [[ ! -f "$target_file" ]]; then
    echo "Target file $target_file does not exist. Creating it."
    touch "$target_file"
  fi

  # Read source content and write to target file with markers
  {
    echo "{{ if .Values.installCRDs }}"
    cat "$source_file"
    echo "{{- end }}"
  } > "$target_file"

  echo "Content from $source_file has been injected into $target_file with markers."
}

# Loop through source and target files
for i in "${!SOURCE_FILES[@]}"; do
  replace_file_content "${SOURCE_FILES[$i]}" "${TARGET_FILES[$i]}"
done
