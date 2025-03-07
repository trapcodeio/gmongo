package gmongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math"
)

type PaginatedMeta struct {
	Total    int `json:"total"`
	PerPage  int `json:"perPage"`
	Page     int `json:"page"`
	LastPage int `json:"lastPage"`
}

type Paginated[T any] struct {
	Meta PaginatedMeta `json:"meta"`
	Data T             `json:"data"`
}

// PaginateAggregate - Paginate aggregate
func (coll *Model[T]) PaginateAggregateWithCountQuery(page int, perPage int, countQuery interface{}, query []interface{}) (*Paginated[any], error) {
	// get total count
	totalCount := int64(0)
	if countQuery != nil {
		count, err := coll.Count(countQuery)
		if err != nil {
			return nil, err
		}

		totalCount = count
	} else {
		count, err := coll.CountAggregate(query)
		if err != nil {
			return nil, err
		}

		totalCount = count
	}

	// if no results
	if totalCount == 0 {
		return &Paginated[any]{
			Meta: PaginatedMeta{
				Total:    0,
				PerPage:  perPage,
				Page:     page,
				LastPage: 0,
			},
			Data: []bson.M{},
		}, nil
	}

	// ceil total/perPage
	lastPage := int(math.Ceil(float64(totalCount) / float64(perPage)))
	skip := (page - 1) * perPage

	// add skip and limit to query
	query = append(query, bson.M{"$skip": skip})
	query = append(query, bson.M{"$limit": perPage})

	// find
	cursor, err := coll.Native().Aggregate(
		context.TODO(),
		query,
	)

	if err != nil {
		return nil, err
	}

	// get results
	var results = make([]bson.M, 0)
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	return &Paginated[any]{
		Meta: PaginatedMeta{
			Total:    int(totalCount),
			PerPage:  perPage,
			Page:     page,
			LastPage: lastPage,
		},
		Data: results,
	}, nil
}

func (coll *Model[T]) PaginateAggregate(page int, perPage int, query []interface{}) (*Paginated[any], error) {
	return coll.PaginateAggregateWithCountQuery(page, perPage, nil, query)
}

// Paginate - Paginate Find
func (coll *Model[T]) Paginate(
	page int,
	perPage int,
	query interface{},
	opts ...*options.FindOptions,
) (*Paginated[any], error) {
	// get total count
	totalCount, err := coll.Native().CountDocuments(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	// if no results
	if totalCount == 0 {
		return &Paginated[any]{
			Meta: PaginatedMeta{
				Total:    0,
				PerPage:  perPage,
				Page:     page,
				LastPage: 0,
			},
			Data: []bson.M{},
		}, nil
	}

	// ceil total/perPage
	lastPage := int(math.Ceil(float64(totalCount) / float64(perPage)))
	skip := (page - 1) * perPage

	// build options
	opts = append(opts, options.Find().SetSkip(int64(skip)).SetLimit(int64(perPage)))

	// find
	cursor, err := coll.Native().Find(context.TODO(), query, opts...)
	if err != nil {
		return nil, err
	}

	// get results
	var results = make([]bson.M, 0)
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	return &Paginated[any]{
		Meta: PaginatedMeta{
			Total:    int(totalCount),
			PerPage:  perPage,
			Page:     page,
			LastPage: lastPage,
		},
		Data: results,
	}, nil
}

type PaginateAggregateOptions struct {
	Match       interface{}
	BeforeLimit []bson.M
	AfterLimit  []bson.M
	Total       *int64
}

// PaginateAggregateRaw - Paginate aggregate raw
// About: This function came about when I noticed the poor performance of the `PaginateAggregate` function when running
// lookups. This is because the limit and skip are applied after the lookup which is not efficient or not always the best
// way to paginate. This function allows you to paginate with the limit and skip applied before the lookup.
func (coll *Model[T]) PaginateAggregateRaw(page int, perPage int, Opt *PaginateAggregateOptions) (*Paginated[any], error) {
	var opt PaginateAggregateOptions
	if Opt != nil {
		opt = *Opt
	}

	if opt.Match == nil {
		opt.Match = bson.M{}
	}

	// get total count
	totalCount := int64(0)
	if opt.Total != nil {
		totalCount = *opt.Total
	} else {
		count, err := coll.Count(opt.Match)
		if err != nil {
			return nil, err
		}

		totalCount = count
	}

	// if no results
	if totalCount == 0 {
		return &Paginated[any]{
			Meta: PaginatedMeta{
				Total:    0,
				PerPage:  perPage,
				Page:     page,
				LastPage: 0,
			},
			Data: []bson.M{},
		}, nil
	}

	// ceil total/perPage
	lastPage := int(math.Ceil(float64(totalCount) / float64(perPage)))
	skip := (page - 1) * perPage

	// add skip and limit to query
	query := make([]bson.M, 0)
	query = append(query, bson.M{"$match": opt.Match})
	query = append(query, opt.BeforeLimit...)
	query = append(query, bson.M{"$skip": skip})
	query = append(query, bson.M{"$limit": perPage})
	query = append(query, opt.AfterLimit...)

	// find
	cursor, err := coll.Native().Aggregate(
		context.TODO(),
		query,
	)

	if err != nil {
		return nil, err
	}

	// get results
	var results = make([]bson.M, 0)
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	return &Paginated[any]{
		Meta: PaginatedMeta{
			Total:    int(totalCount),
			PerPage:  perPage,
			Page:     page,
			LastPage: lastPage,
		},
		Data: results,
	}, nil
}
