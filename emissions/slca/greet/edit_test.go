package greet

import (
	"testing"

	"github.com/kr/pretty"
)

func TestEdit(t *testing.T) {
	testdb := &DB{
		Data: &Data{
			StationaryProcesses: []*StationaryProcess{
				&StationaryProcess{
					ID:   "12345",
					Name: "Goat Farming",
					Inputs: []*Input{
						&Input{
							ID:     "248694",
							Source: "Well",
							AmountYears: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "100"}.encode(),
								},
							},
						},
						&Input{
							ID:     "2486948",
							Source: "Well",
							AmountYears: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "100"}.encode(),
								},
							},
						},
					},
					InputGroups: []*InputGroup{
						&InputGroup{
							Type: "amount",
							Amount: []*ValueYear{
								&ValueYear{
									Value: expression{val: "200"}.encode(),
									Year:  "0",
								},
							},
						},
					},
					OtherEmissions: []*OtherEmission{
						&OtherEmission{
							Ref: "abcdefg",
							ValueYears: []*ValueYear{
								&ValueYear{
									Value: expression{val: "200"}.encode(),
									Year:  "0",
								},
							},
						},
						&OtherEmission{
							Ref: "gfedcba",
							ValueYears: []*ValueYear{
								&ValueYear{
									Value: expression{val: "300"}.encode(),
									Year:  "0",
								},
							},
						},
					},
				},
				&StationaryProcess{
					ID:   "123456",
					Name: "Goat Farming",
					Inputs: []*Input{
						&Input{
							ID:     "2486942",
							Source: "Well",
							AmountYears: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "100"}.encode(),
								},
							},
						},
					},
				},
			},
			Mixes: []*Mix{
				&Mix{
					ID: "12345",
					PathwayRefs: []*MixPathway{
						&MixPathway{
							Ref: "abcdefg",
							Shares: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "111"}.encode(),
								},
								&ValueYear{
									Year:  "1",
									Value: expression{val: "222"}.encode(),
								},
							},
						},
						&MixPathway{
							Ref: "gfedbca",
							Shares: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "111"}.encode(),
								},
								&ValueYear{
									Year:  "1",
									Value: expression{val: "222"}.encode(),
								},
							},
						},
					},
				},
				&Mix{
					ID: "54321",
					PathwayRefs: []*MixPathway{
						&MixPathway{
							Ref: "abcdefg",
							Shares: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "999"}.encode(),
								},
							},
						},
					},
				},
			},
		},
	}

	editdb := &DB{
		Data: &Data{
			StationaryProcesses: []*StationaryProcess{
				&StationaryProcess{
					ID:   "12345",
					Name: "Goat Farming edited",
					Inputs: []*Input{
						&Input{
							ID:     "248694",
							Source: "Pathway",
							AmountYears: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "50"}.encode(),
								},
							},
						},
					},
					InputGroups: []*InputGroup{
						&InputGroup{
							Type: "amount",
							Amount: []*ValueYear{
								&ValueYear{
									Value: expression{val: "400"}.encode(),
									Year:  "0",
								},
							},
						},
					},
					OtherEmissions: []*OtherEmission{
						&OtherEmission{
							Ref: "abcdefg",
							ValueYears: []*ValueYear{
								&ValueYear{
									Value: expression{val: "800"}.encode(),
									Year:  "0",
								},
							},
						},
					},
				},
			},
			Mixes: []*Mix{
				&Mix{
					ID: "12345",
					PathwayRefs: []*MixPathway{
						&MixPathway{
							Ref: "abcdefg",
							Shares: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "666"}.encode(),
								},
								&ValueYear{
									Year:  "1",
									Value: expression{val: "777"}.encode(),
								},
							},
						},
					},
				},
			},
		},
	}

	resultdb := &DB{
		Data: &Data{
			StationaryProcesses: []*StationaryProcess{
				&StationaryProcess{
					ID:   "12345",
					Name: "Goat Farming edited",
					Inputs: []*Input{
						&Input{
							ID:     "248694",
							Source: "Pathway",
							AmountYears: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "50"}.encode(),
								},
							},
						},
						&Input{
							ID:     "2486948",
							Source: "Well",
							AmountYears: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "100"}.encode(),
								},
							},
						},
					},
					InputGroups: []*InputGroup{
						&InputGroup{
							Type: "amount",
							Amount: []*ValueYear{
								&ValueYear{
									Value: expression{val: "400"}.encode(),
									Year:  "0",
								},
							},
						},
					},
					OtherEmissions: []*OtherEmission{
						&OtherEmission{
							Ref: "abcdefg",
							ValueYears: []*ValueYear{
								&ValueYear{
									Value: expression{val: "800"}.encode(),
									Year:  "0",
								},
							},
						},
						&OtherEmission{
							Ref: "gfedcba",
							ValueYears: []*ValueYear{
								&ValueYear{
									Value: expression{val: "300"}.encode(),
									Year:  "0",
								},
							},
						},
					},
				},
				&StationaryProcess{
					ID:   "123456",
					Name: "Goat Farming",
					Inputs: []*Input{
						&Input{
							ID:     "2486942",
							Source: "Well",
							AmountYears: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "100"}.encode(),
								},
							},
						},
					},
				},
			},
			Mixes: []*Mix{
				&Mix{
					ID: "12345",
					PathwayRefs: []*MixPathway{
						&MixPathway{
							Ref: "abcdefg",
							Shares: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "666"}.encode(),
								},
								&ValueYear{
									Year:  "1",
									Value: expression{val: "777"}.encode(),
								},
							},
						},
						&MixPathway{
							Ref: "gfedbca",
							Shares: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "111"}.encode(),
								},
								&ValueYear{
									Year:  "1",
									Value: expression{val: "222"}.encode(),
								},
							},
						},
					},
				},
				&Mix{
					ID: "54321",
					PathwayRefs: []*MixPathway{
						&MixPathway{
							Ref: "abcdefg",
							Shares: []*ValueYear{
								&ValueYear{
									Year:  "0",
									Value: expression{val: "999"}.encode(),
								},
							},
						},
					},
				},
			},
		},
	}

	testdb.EditByID(editdb)
	diff := pretty.Diff(testdb, resultdb)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestEditExpression(t *testing.T) {
	testdb := &DB{
		Data: &Data{
			StationaryProcesses: []*StationaryProcess{
				&StationaryProcess{
					Inputs: []*Input{
						&Input{
							AmountYears: []*ValueYear{
								&ValueYear{
									Value: expression{
										val:   "100",
										names: []string{"abc", "def"},
									}.encode(),
								},
								&ValueYear{
									Value: expression{
										val:   "100",
										names: []string{"qqq", "vvv"},
									}.encode(),
								},
								&ValueYear{
									Value: expression{
										val:   "100",
										names: []string{"vvv", "xxx"},
									}.encode(),
								},
							},
						},
					},
				},
			},
		},
	}

	resultdb := &DB{
		Data: &Data{
			StationaryProcesses: []*StationaryProcess{
				&StationaryProcess{
					Inputs: []*Input{
						&Input{
							AmountYears: []*ValueYear{
								&ValueYear{
									Value: expression{
										val:   "200",
										names: []string{"abc", "def"},
									}.encode(),
								},
								&ValueYear{
									Value: expression{
										val:   "100",
										names: []string{"qqq", "vvv"},
									}.encode(),
								},
								&ValueYear{
									Value: expression{
										val:   "100",
										names: []string{"vvv", "xxx"},
									}.encode(),
								},
							},
						},
					},
				},
			},
		},
	}
	err := testdb.EditExpressionByID("abc", 200.)
	if err != nil {
		t.Fatal(err)
	}
	diff := pretty.Diff(testdb, resultdb)
	if len(diff) != 0 {
		t.Fatal(diff)
	}

	// This should fail
	err = testdb.EditExpressionByID("vvv", 200.)
	expected := "multiple matches for expression vvv"
	if err.Error() != expected {
		t.Fatalf("expected error `%s`; got `%s`", expected, err.Error())
	}

	err = testdb.EditExpressionByID("ppp", 200.)
	expected = "could not find expression ppp"
	if err.Error() != expected {
		t.Fatalf("expected error `%s`; got `%s`", expected, err.Error())
	}

}
