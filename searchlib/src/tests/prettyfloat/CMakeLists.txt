# Copyright Yahoo. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
vespa_add_executable(searchlib_prettyfloat_test_app TEST
    SOURCES
    prettyfloat.cpp
    DEPENDS
    searchlib
)
vespa_add_test(NAME searchlib_prettyfloat_test_app COMMAND searchlib_prettyfloat_test_app)
