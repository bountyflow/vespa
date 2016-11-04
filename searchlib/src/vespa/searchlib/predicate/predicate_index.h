// Copyright 2016 Yahoo Inc. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.

#pragma once

#include "document_features_store.h"
#include "predicate_interval_store.h"
#include "simple_index.h"
#include <vespa/searchlib/common/bitvectorcache.h>
#include <unordered_map>
#include <vespa/vespalib/data/databuffer.h>
#include <vespa/vespalib/stllike/string.h>
#include "predicate_interval.h"

namespace search {
namespace predicate {
class PredicateTreeAnnotations;

/**
 * PredicateIndex keeps an index of boolean constraints for use with
 * the interval algorithm. It is the central component of
 * PredicateAttribute, and PredicateBlueprint uses it to obtain
 * posting lists for matching.
 */
class PredicateIndex : public PopulateInterface {
    typedef SimpleIndex<datastore::EntryRef> IntervalIndex;
    typedef SimpleIndex<datastore::EntryRef> BoundsIndex;
    typedef btree::BTree<uint32_t, btree::BTreeNoLeafData> BTreeSet;
    template <typename IntervalT>
    using FeatureMap = std::unordered_map<uint64_t, std::vector<IntervalT>>;
    using generation_t = vespalib::GenerationHandler::generation_t;
    template <typename T>
    using optional = std::experimental::optional<T>;

public:
    using ZeroConstraintDocs = BTreeSet::FrozenView;
    typedef std::unique_ptr<PredicateIndex> UP;
    typedef vespalib::GenerationHandler GenerationHandler;
    typedef vespalib::GenerationHolder GenerationHolder;
    using BTreeIterator = SimpleIndex<datastore::EntryRef>::BTreeIterator;
    using VectorIterator = SimpleIndex<datastore::EntryRef>::VectorIterator;
    static const vespalib::string z_star_attribute_name;
    static const uint64_t z_star_hash;
    static const vespalib::string z_star_compressed_attribute_name;
    static const uint64_t z_star_compressed_hash;

private:
    uint32_t _arity;
    GenerationHandler &_generation_handler;
    const DocIdLimitProvider &_limit_provider;
    IntervalIndex _interval_index;
    BoundsIndex _bounds_index;
    PredicateIntervalStore _interval_store;
    BTreeSet _zero_constraint_docs;

    DocumentFeaturesStore _features_store;
    mutable BitVectorCache  _cache;

    template <typename IntervalT>
    void addPosting(uint64_t feature, uint32_t doc_id,
                    datastore::EntryRef ref);

    template <typename IntervalT>
    void indexDocumentFeatures(uint32_t doc_id, const FeatureMap<IntervalT> &interval_map);

    PopulateInterface::Iterator::UP lookup(uint64_t key) const override;

public:
    PredicateIndex(GenerationHandler &generation_handler, GenerationHolder &genHolder,
                   const DocIdLimitProvider &limit_provider,
                   const SimpleIndexConfig &simple_index_config, uint32_t arity)
        : _arity(arity),
          _generation_handler(generation_handler),
          _limit_provider(limit_provider),
          _interval_index(genHolder, limit_provider, simple_index_config),
          _bounds_index(genHolder, limit_provider, simple_index_config),
          _interval_store(),
          _zero_constraint_docs(),
          _features_store(arity),
          _cache(genHolder)  {
    }
    // deserializes PredicateIndex from buffer.
    // The observer can be used to gain some insight into what has been added to the index..
    PredicateIndex(GenerationHandler &generation_handler, GenerationHolder &genHolder,
                   const DocIdLimitProvider &limit_provider,
                   const SimpleIndexConfig &simple_index_config, vespalib::DataBuffer &buffer,
                   SimpleIndexDeserializeObserver<> & observer, uint32_t version);

    void serialize(vespalib::DataBuffer &buffer) const;
    void onDeserializationCompleted();

    void indexEmptyDocument(uint32_t doc_id);
    void indexDocument(uint32_t doc_id, const PredicateTreeAnnotations &annotations);
    void removeDocument(uint32_t doc_id);
    void commit();
    void trimHoldLists(generation_t used_generation);
    void transferHoldLists(generation_t generation);
    MemoryUsage getMemoryUsage() const;

    int getArity() const { return _arity; }

    const ZeroConstraintDocs getZeroConstraintDocs() const {
        return _zero_constraint_docs.getFrozenView();
    }

    const IntervalIndex &getIntervalIndex() const {
        return _interval_index;
    }

    const BoundsIndex &getBoundsIndex() const {
        return _bounds_index;
    }

    const PredicateIntervalStore &getIntervalStore() const {
        return _interval_store;
    }

    void populateIfNeeded(size_t doc_id_limit);
    BitVectorCache::KeySet lookupCachedSet(const BitVectorCache::KeyAndCountSet & keys) const;
    void computeCountVector(BitVectorCache::KeySet & keys, BitVectorCache::CountVector & v) const;

    /*
     * Adjust size of structures to have space for docId.
     */
    void adjustDocIdLimit(uint32_t docId);
};

extern template class SimpleIndex<datastore::EntryRef>;

}  // namespace predicate
}  // namespace search

