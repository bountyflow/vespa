// Copyright 2016 Yahoo Inc. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.

#include <vespa/fastos/fastos.h>
#include "btreenodestore.hpp"
#include "btreenode.h"
#include "btreerootbase.h"
#include "btreeroot.h"
#include "btreenodeallocator.h"
#include <vespa/searchlib/datastore/datastore.h>

namespace search
{

namespace btree
{

template class BTreeNodeStore<uint32_t, uint32_t,
                              NoAggregated,
                              BTreeDefaultTraits::INTERNAL_SLOTS,
                              BTreeDefaultTraits::LEAF_SLOTS>;
template class BTreeNodeStore<uint32_t, BTreeNoLeafData,
                              NoAggregated,
                              BTreeDefaultTraits::INTERNAL_SLOTS,
                              BTreeDefaultTraits::LEAF_SLOTS>;
template class BTreeNodeStore<uint32_t, int32_t,
                              MinMaxAggregated,
                              BTreeDefaultTraits::INTERNAL_SLOTS,
                              BTreeDefaultTraits::LEAF_SLOTS>;


typedef BTreeNodeStore<uint32_t, uint32_t, NoAggregated,
                       BTreeDefaultTraits::INTERNAL_SLOTS,
                       BTreeDefaultTraits::LEAF_SLOTS>        MyNodeStore1;
typedef BTreeNodeStore<uint32_t, BTreeNoLeafData, NoAggregated,
                       BTreeDefaultTraits::INTERNAL_SLOTS,
                       BTreeDefaultTraits::LEAF_SLOTS> MyNodeStore2;
typedef BTreeNodeStore<uint32_t, int32_t, MinMaxAggregated,
                       BTreeDefaultTraits::INTERNAL_SLOTS,
                       BTreeDefaultTraits::LEAF_SLOTS>        MyNodeStore3;

} // namespace btree

namespace datastore {

typedef EntryRefT<22> MyRef;
typedef btree::BTreeLeafNode<uint32_t, uint32_t, btree::NoAggregated>         MyEntry1;
typedef btree::BTreeLeafNode<uint32_t, btree::BTreeNoLeafData, btree::NoAggregated>  MyEntry2;
typedef btree::BTreeInternalNode<uint32_t, btree::NoAggregated>               MyEntry4;
typedef btree::BTreeLeafNode<uint32_t, int32_t, btree::MinMaxAggregated>     MyEntry5;
typedef btree::BTreeInternalNode<uint32_t, btree::MinMaxAggregated>           MyEntry6;

template
std::pair<MyRef, MyEntry1 *>
datastore::DataStoreT<MyRef>::allocNewEntryCopy<MyEntry1>(uint32_t, const MyEntry1 &);

template
std::pair<MyRef, MyEntry2 *>
datastore::DataStoreT<MyRef>::allocNewEntryCopy<MyEntry2>(uint32_t, const MyEntry2 &);

template
std::pair<MyRef, MyEntry4 *>
datastore::DataStoreT<MyRef>::allocNewEntryCopy<MyEntry4>(uint32_t, const MyEntry4 &);

template
std::pair<MyRef, MyEntry5 *>
datastore::DataStoreT<MyRef>::allocNewEntryCopy<MyEntry5>(uint32_t, const MyEntry5 &);

template
std::pair<MyRef, MyEntry6 *>
datastore::DataStoreT<MyRef>::allocNewEntryCopy<MyEntry6>(uint32_t, const MyEntry6 &);

template
std::pair<MyRef, MyEntry1 *>
datastore::DataStoreT<MyRef>::allocEntry<MyEntry1, btree::BTreeNodeReclaimer>(uint32_t);

template
std::pair<MyRef, MyEntry2 *>
datastore::DataStoreT<MyRef>::allocEntry<MyEntry2, btree::BTreeNodeReclaimer>(uint32_t);

template
std::pair<MyRef, MyEntry4 *>
datastore::DataStoreT<MyRef>::allocEntry<MyEntry4, btree::BTreeNodeReclaimer>(uint32_t);

template
std::pair<MyRef, MyEntry5 *>
datastore::DataStoreT<MyRef>::allocEntry<MyEntry5, btree::BTreeNodeReclaimer>(uint32_t);

template
std::pair<MyRef, MyEntry6 *>
datastore::DataStoreT<MyRef>::allocEntry<MyEntry6, btree::BTreeNodeReclaimer>(uint32_t);

template
std::pair<MyRef, MyEntry1 *>
datastore::DataStoreT<MyRef>::allocEntryCopy<MyEntry1, btree::BTreeNodeReclaimer>(
        uint32_t, const MyEntry1 &);

template
std::pair<MyRef, MyEntry2 *>
datastore::DataStoreT<MyRef>::allocEntryCopy<MyEntry2, btree::BTreeNodeReclaimer>(
        uint32_t, const MyEntry2 &);

template
std::pair<MyRef, MyEntry4 *>
datastore::DataStoreT<MyRef>::allocEntryCopy<MyEntry4, btree::BTreeNodeReclaimer>(
        uint32_t, const MyEntry4 &);

template
std::pair<MyRef, MyEntry5 *>
datastore::DataStoreT<MyRef>::allocEntryCopy<MyEntry5, btree::BTreeNodeReclaimer>(
        uint32_t, const MyEntry5 &);


template
std::pair<MyRef, MyEntry6 *>
datastore::DataStoreT<MyRef>::allocEntryCopy<MyEntry6, btree::BTreeNodeReclaimer>(
        uint32_t, const MyEntry6 &);



} // namespace btree

} // namespace search
