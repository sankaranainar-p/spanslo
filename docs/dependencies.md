# Dependencies

| Name | Version | License | Why |
| --- | --- | --- | --- |
| jsonschema | 4.26.0 | MIT | Schema tests for `spec/` (bench dev group); pulls in attrs, jsonschema-specifications, referencing, rpds-py (all MIT), typing-extensions (PSF-2.0) |
| PyYAML | 6.0.3 | MIT | Parse catalog examples and `spec/semconv-mapping.yaml` in tests (bench dev group) |
| go.yaml.in/yaml/v3 | v3.0.6 | MIT and Apache-2.0 (dual: MIT for the libyaml port files, Apache-2.0 for the rest) | Read `spec/semconv-mapping.yaml` and catalog files in Go (core) |