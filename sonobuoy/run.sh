#!/bin/bash

set -x

# Define the directory for test results, defaulting to /tmp/results
results_dir="${RESULTS_DIR:-/tmp/results}"
mkdir -p ${results_dir}

# Function to package results and signal Sonobuoy
saveResults() {
    cd ${results_dir}

    # Package the results into a tarball for Sonobuoy
    tar czf results.tar.gz *

    # Signal Sonobuoy by writing the location of results
    printf ${results_dir}/results.tar.gz > ${results_dir}/done
}

# Ensure that the saveResults function runs upon exit
trap saveResults EXIT

# Run the Ginkgo test suite
ginkgo run -r --keep-going --output-dir=${results_dir} --junit-report=junit.xml -p /workspace/tests &>${results_dir}/out
