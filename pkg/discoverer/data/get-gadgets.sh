#!/bin/bash

# Fetch gadgets data and save to file
echo "Fetching gadgets data from ArtifactHub..."
curl -X 'GET' 'https://artifacthub.io/api/v1/packages/search?kind=22&limit=60&verified_publisher=true&official=true&cncf=true' \
  -H 'accept: application/json' \
  -o gadgets-full.json

# Check if the API call was successful
if [ $? -eq 0 ] && [ -f gadgets-full.json ]; then
    echo "Full data saved to gadgets-full.json"

    # Extract normalized names for detailed API calls
    normalized_names=$(jq -r '.packages[].normalized_name' gadgets-full.json)

    echo "Fetching detailed information for each gadget..."

    # Create temporary file for detailed data
    echo '{"packages":[]}' > gadgets-detailed.json

    # Counter for progress
    total=$(echo "$normalized_names" | wc -l)
    current=0

    # Fetch detailed info for each gadget
    for name in $normalized_names; do
        current=$((current + 1))
        echo "Processing $current/$total: $name"

        # Fetch detailed gadget info
        curl -s -X 'GET' "https://artifacthub.io/api/v1/packages/inspektor-gadget/gadgets/$name" \
          -H 'accept: application/json' \
          -o "temp-$name.json"

        if [ -f "temp-$name.json" ] && [ -s "temp-$name.json" ]; then
            # Extract container images from the detailed response
            container_images=$(jq -r '.containers_images[]?.image // empty' "temp-$name.json" 2>/dev/null | jq -R -s 'split("\n") | map(select(length > 0))')

            # Get basic info from original response
            basic_info=$(jq --arg name "$name" '.packages[] | select(.normalized_name == $name) | {normalized_name: .normalized_name, description: .description}' gadgets-full.json)

            # Combine basic info with container image (pick first one)
            if [ "$container_images" != "null" ] && [ "$container_images" != "[]" ]; then
                combined=$(echo "$basic_info" | jq --argjson images "$container_images" '. + {container_image: ($images[0] // null)}')
            else
                combined=$(echo "$basic_info" | jq '. + {container_image: null}')
            fi

            # Add to the detailed json array
            jq --argjson item "$combined" '.packages += [$item]' gadgets-detailed.json > temp-combined.json && mv temp-combined.json gadgets-detailed.json
        fi

        # Clean up temp file
        rm -f "temp-$name.json"
    done

    # Move the detailed file to final location
    mv gadgets-detailed.json gadgets.json

    # Remove the full data file after filtering
    rm gadgets-full.json
    echo "Temporary full data file removed"

    echo "Enhanced data saved to gadgets.json"
    echo "Preview of enhanced data:"
    jq '.packages[:3]' gadgets.json
else
    echo "Error: Failed to fetch data from API"
    exit 1
fi
