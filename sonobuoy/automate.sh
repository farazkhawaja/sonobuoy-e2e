#!/bin/bash

## Function to install Sonobuoy if it is not installed
install_sonobuoy() {
    if ! command -v sonobuoy &> /dev/null
    then
        echo "Sonobuoy not found, installing..."
        curl -L https://github.com/vmware-tanzu/sonobuoy/releases/download/v0.56.12/sonobuoy_0.56.12_linux_amd64.tar.gz | tar -xz
        cp sonobuoy /usr/local/bin/
        cp sonobuoy /usr/bin
        echo "Sonobuoy installed successfully."
    else
        echo "Sonobuoy is already installed."
    fi
}

# Function to clean up existing Sonobuoy resources
cleanup_sonobuoy() {
    echo "Deleting all Sonobuoy resources..."
    sonobuoy delete --all --wait
    echo "Sonobuoy resources deleted."
}

# Function to generate the Sonobuoy plugin YAML
generate_plugin_yaml() {
    echo "Generating Sonobuoy plugin YAML..."
    sonobuoy gen plugin --name=faraz-e2e --image=khwajafaraz/sonobuoy-e2e:latest --env TEST_NAMESPACE="install-namespace" --show-default-podspec > faraz-e2e-plugin.yaml
    echo "Plugin YAML generated: faraz-e2e-plugin.yaml"
}

# Function to run Sonobuoy
run_sonobuoy() {
    echo "Running Sonobuoy with custom plugin..."
    sonobuoy run --plugin ./faraz-e2e-plugin.yaml --wait
    echo "Sonobuoy run complete."
}

# Function to retrieve the results in a tar file provided as an argument
retrieve_results() {
    result_dir=$1
    echo "Retrieving Sonobuoy results..."
    sonobuoy retrieve ./$result_dir
    echo "Results retrieved: $result_dir"
}

# Function to extract and display the 'out' file
extract_and_display_out() {
    result_dir=$1
    echo "Finding the actual tarball inside $result_dir directory..."

    # Navigate to the directory and find the actual tarball
    cd $result_dir
    actual_tarball=$(find . -name '*.tar.gz')

    if [ -z "$actual_tarball" ]; then
        echo "No tarball found inside the $result_dir directory."
        exit 1
    fi

    echo "Extracting and displaying 'out' file from $actual_tarball..."
    tar -zxvf $actual_tarball --wildcards '*/out'

    # Display the contents of the 'out' file
    cat plugins/faraz-e2e/results/global/out
    cd ..
}
# Main script starts here

# Check if Sonobuoy is installed, and install it if not
install_sonobuoy

# Clean up existing Sonobuoy resources
cleanup_sonobuoy

# Generate Sonobuoy plugin YAML
generate_plugin_yaml

# Run Sonobuoy
run_sonobuoy

# Retrieve results (pass the tar filename as an argument to the script)
if [ -z "$1" ]; then
    echo "Please provide a dir name as an argument to the script."
    exit 1
fi
retrieve_results $1

# Extract and display the 'out' file
extract_and_display_out $1

#FutureTODOs
#mount pvc to container and crud ops for a file
#volumeclaimtemplate for pvc pv
#dont exec into test pvc pod but use job to get failure /success status for running crud commands