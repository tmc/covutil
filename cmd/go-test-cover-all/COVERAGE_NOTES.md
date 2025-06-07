# Coverage Collection Notes

## GOCOVERDIR Investigation Results

After extensive testing, we found that GOCOVERDIR is not producing coverage files in our environment (macOS, Go 1.24.3). Testing showed:

1. **GOCOVERDIR environment variable**: No coverage files generated
2. **-args -test.gocoverdir**: No coverage files generated  
3. **Building with -cover and running with GOCOVERDIR**: Shows coverage percentage but no files
4. **Both approaches combined**: Still no coverage files

The tests confirm that coverage is being calculated (we see percentages like "coverage: 100.0% of statements") but the coverage data files (covcounters.* and covmeta.*) are not being written to the specified directory.

## Current Implementation

Due to the GOCOVERDIR issues, the tool uses the traditional `-coverprofile` approach:

1. Each module is tested with `-coverprofile=<file>`
2. Coverage profiles are collected in `.coverage/profiles/`
3. Profiles are merged manually by parsing and combining the coverage data
4. The merged profile can be analyzed with standard `go tool cover` commands

## Known Issues

1. **Module path handling**: Coverage profiles use module-relative paths which can cause issues when merging profiles from different modules
2. **Coverage merge**: The current merge implementation may not handle all edge cases correctly
3. **HTML report generation**: May fail if the merged profile has path resolution issues

## Future Improvements

1. Investigate GOCOVERDIR support on different platforms
2. Improve module path handling in coverage merge
3. Add support for converting traditional profiles to GOCOVERDIR format when it becomes more stable
4. Consider using `go tool cover -html` with module-specific working directories