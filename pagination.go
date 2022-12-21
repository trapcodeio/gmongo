package gmongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
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
