# Copyright Yahoo. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
vespa_define_module(
    DEPENDS
    vespalog
    vespalib
    configdefinitions
    document
    searchlib

    LIBS
    src/vespa/juniper
    src/vespa/searchsummary
    src/vespa/searchsummary/config
    src/vespa/searchsummary/docsummary
    src/vespa/searchsummary/test

    TESTS
    src/tests/docsumformat
    src/tests/docsummary
    src/tests/docsummary/attribute_combiner
    src/tests/docsummary/attributedfw
    src/tests/docsummary/document_id_dfw
    src/tests/docsummary/matched_elements_filter
    src/tests/docsummary/slime_summary
    src/tests/docsummary/summary_field_converter
    src/tests/extractkeywords
    src/tests/juniper
)
