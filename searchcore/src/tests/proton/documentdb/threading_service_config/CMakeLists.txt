# Copyright Yahoo. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
vespa_add_executable(searchcore_threading_service_config_test_app TEST
    SOURCES
    threading_service_config_test.cpp
    DEPENDS
    searchcore_server
)
vespa_add_test(NAME searchcore_threading_service_config_test_app COMMAND searchcore_threading_service_config_test_app)
