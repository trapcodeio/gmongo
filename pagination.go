package gmongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math"
)

type PaginatedMeta struct {
	Total    int `json:"total"`
	PerPage  int `json:"per_page"`
	Page     int `json:"page"`
	LastPage int `json:"last_page"`
}

type Paginated[T any] struct {
	Meta PaginatedMeta `json:"meta"`
	Data T             `json:"data"`
}

// PaginateAggregate - Paginate aggregate
func (coll *Model[T]) PaginateAggregate(page int, perPage int, query []interface{}) (*Paginated[any], error) {
	// get total count
	totalCount, err := coll.CountAggregate(query)
	if err != nil {
		return nil, err
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
			Total:    totalCount,
			PerPage:  perPage,
			Page:     page,
			LastPage: lastPage,
		},
		Data: results,
	}, nil
}

// Paginate - Paginate Find
func (coll *Model[T]) Paginate(
	page int,
	perPage int,
	query bson.M,
	opts ...*options.FindOptions,
) (*Paginated[any], error) {
	// get total count
	totalCount, err := coll.Native().CountDocuments(context.TODO(), query)
	if err != nil {
		return nil, err
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
