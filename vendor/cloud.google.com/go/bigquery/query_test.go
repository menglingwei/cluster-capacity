// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bigquery

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	bq "google.golang.org/api/bigquery/v2"
)

func defaultQueryJob() *bq.Job {
	return &bq.Job{
		Configuration: &bq.JobConfiguration{
			Query: &bq.JobConfigurationQuery{
				DestinationTable: &bq.TableReference{
					ProjectId: "project-id",
					DatasetId: "dataset-id",
					TableId:   "table-id",
				},
				Query: "query string",
				DefaultDataset: &bq.DatasetReference{
					ProjectId: "def-project-id",
					DatasetId: "def-dataset-id",
				},
			},
		},
	}
}

func TestQuery(t *testing.T) {
	s := &testService{}
	c := &Client{
		projectID: "project-id",
		service:   s,
	}
	testCases := []struct {
		dst     *Table
		src     *Query
		options []Option
		want    *bq.Job
	}{
		{
			dst:  c.Dataset("dataset-id").Table("table-id"),
			src:  defaultQuery,
			want: defaultQueryJob(),
		},
		{
			dst: c.Dataset("dataset-id").Table("table-id"),
			src: &Query{
				Q: "query string",
			},
			want: func() *bq.Job {
				j := defaultQueryJob()
				j.Configuration.Query.DefaultDataset = nil
				return j
			}(),
		},
		{
			dst: &Table{},
			src: defaultQuery,
			want: func() *bq.Job {
				j := defaultQueryJob()
				j.Configuration.Query.DestinationTable = nil
				return j
			}(),
		},
		{
			dst: &Table{
				ProjectID: "project-id",
				DatasetID: "dataset-id",
				TableID:   "table-id",
			},
			src:     defaultQuery,
			options: []Option{CreateNever, WriteTruncate},
			want: func() *bq.Job {
				j := defaultQueryJob()
				j.Configuration.Query.WriteDisposition = "WRITE_TRUNCATE"
				j.Configuration.Query.CreateDisposition = "CREATE_NEVER"
				return j
			}(),
		},
		{
			dst:     c.Dataset("dataset-id").Table("table-id"),
			src:     defaultQuery,
			options: []Option{DisableQueryCache()},
			want: func() *bq.Job {
				j := defaultQueryJob()
				f := false
				j.Configuration.Query.UseQueryCache = &f
				return j
			}(),
		},
		{
			dst:     c.Dataset("dataset-id").Table("table-id"),
			src:     defaultQuery,
			options: []Option{AllowLargeResults()},
			want: func() *bq.Job {
				j := defaultQueryJob()
				j.Configuration.Query.AllowLargeResults = true
				return j
			}(),
		},
		{
			dst:     c.Dataset("dataset-id").Table("table-id"),
			src:     defaultQuery,
			options: []Option{DisableFlattenedResults()},
			want: func() *bq.Job {
				j := defaultQueryJob()
				f := false
				j.Configuration.Query.FlattenResults = &f
				j.Configuration.Query.AllowLargeResults = true
				return j
			}(),
		},
		{
			dst:     c.Dataset("dataset-id").Table("table-id"),
			src:     defaultQuery,
			options: []Option{JobPriority("low")},
			want: func() *bq.Job {
				j := defaultQueryJob()
				j.Configuration.Query.Priority = "low"
				return j
			}(),
		},
		{
			dst:     c.Dataset("dataset-id").Table("table-id"),
			src:     defaultQuery,
			options: []Option{MaxBillingTier(3), MaxBytesBilled(5)},
			want: func() *bq.Job {
				j := defaultQueryJob()
				tier := int64(3)
				j.Configuration.Query.MaximumBillingTier = &tier
				j.Configuration.Query.MaximumBytesBilled = 5
				return j
			}(),
		},
		{
			dst:     c.Dataset("dataset-id").Table("table-id"),
			src:     defaultQuery,
			options: []Option{MaxBytesBilled(-1)},
			want:    defaultQueryJob(),
		},
		{
			dst:     c.Dataset("dataset-id").Table("table-id"),
			src:     defaultQuery,
			options: []Option{QueryUseStandardSQL()},
			want: func() *bq.Job {
				j := defaultQueryJob()
				j.Configuration.Query.UseLegacySql = false
				j.Configuration.Query.ForceSendFields = []string{"UseLegacySql"}
				return j
			}(),
		},
		{
			dst: c.Dataset("dataset-id").Table("table-id"),
			src: &Query{
				Q: "query string",
				TableDefinitions: map[string]ExternalData{
					"atable": &GCSReference{
						uris:                []string{"uri"},
						AllowJaggedRows:     true,
						AllowQuotedNewlines: true,
						Compression:         Gzip,
						Encoding:            UTF_8,
						FieldDelimiter:      ";",
						IgnoreUnknownValues: true,
						MaxBadRecords:       1,
						Quote:               "'",
						SkipLeadingRows:     2,
						Schema: Schema([]*FieldSchema{
							{Name: "name", Type: StringFieldType},
						}),
					},
				},
			},
			want: func() *bq.Job {
				j := defaultQueryJob()
				j.Configuration.Query.DefaultDataset = nil
				td := make(map[string]bq.ExternalDataConfiguration)
				quote := "'"
				td["atable"] = bq.ExternalDataConfiguration{
					Compression:         "GZIP",
					IgnoreUnknownValues: true,
					MaxBadRecords:       1,
					SourceFormat:        "CSV", // must be explicitly set.
					SourceUris:          []string{"uri"},
					CsvOptions: &bq.CsvOptions{
						AllowJaggedRows:     true,
						AllowQuotedNewlines: true,
						Encoding:            "UTF-8",
						FieldDelimiter:      ";",
						SkipLeadingRows:     2,
						Quote:               &quote,
					},
					Schema: &bq.TableSchema{
						Fields: []*bq.TableFieldSchema{
							{Name: "name", Type: "STRING"},
						},
					},
				}
				j.Configuration.Query.TableDefinitions = td
				return j
			}(),
		},
	}

	for _, tc := range testCases {
		if _, err := c.Copy(context.Background(), tc.dst, tc.src, tc.options...); err != nil {
			t.Errorf("err calling query: %v", err)
			continue
		}
		if !reflect.DeepEqual(s.Job, tc.want) {
			t.Errorf("querying: got:\n%v\nwant:\n%v", s.Job, tc.want)
		}
	}
}
