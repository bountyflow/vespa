// Copyright Yahoo. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.

#include "documentlocations.h"
#include <vespa/searchlib/attribute/attributeguard.h>
#include <vespa/searchlib/attribute/attributevector.h>

namespace search {
namespace common {

DocumentLocations::DocumentLocations()
    : _vec(NULL)
{
}

DocumentLocations::~DocumentLocations() { }

DocumentLocations::DocumentLocations(DocumentLocations &&) = default;
DocumentLocations & DocumentLocations::operator = (DocumentLocations &&) = default;


}  // namespace common
}  // namespace search
