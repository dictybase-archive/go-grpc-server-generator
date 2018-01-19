package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/urfave/cli.v1"

	. "github.com/dave/jennifer/jen"
)

func main() {
	app := cli.NewApp()
	app.Version = "1.0.0"
	app.Name = "go-grpc-server-generator"
	app.Usage = "Generates code for implementing grpc server"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "service,s",
			Usage: "service name",
		},
		cli.StringFlag{
			Name:  "short-service,ss",
			Usage: "short name of the service",
		},
		cli.StringFlag{
			Name:  "package,p",
			Usage: "package name",
			Value: "server",
		},
		cli.StringFlag{
			Name:  "output,o",
			Usage: "output file name, by default printed to stdout",
		},
	}
	app.Before = validateParams
	app.Action = generateCode
	app.Run(os.Args)
}

func validateParams(c *cli.Context) error {
	for _, p := range []string{"service", "short-service"} {
		if len(c.String(p)) == 0 {
			return cli.NewExitError(
				fmt.Sprintf("option %s is not set", p),
				2,
			)
		}
	}
	return nil
}

func generateCode(c *cli.Context) error {
	var output io.Writer
	if len(c.String("output")) > 0 {
		w, err := os.Create(c.String("output"))
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("unable to open file %s", c.String("output")),
				2,
			)
		}
		output = w
		defer w.Close()
	} else {
		output = os.Stdout
	}

	f := NewFile(c.String("package"))

	shortSrv := c.String("short-service")
	srv := c.String("service")
	usrv := strings.Title(srv)
	srvPayload := fmt.Sprintf("%s.%s", srv, usrv)
	srvPath := fmt.Sprintf("%s/%s", "github.com/dictyBase/go-genproto/dictybaseapis", srv)
	srvAttr := fmt.Sprintf("%s.%sAttributes", srv, usrv)
	srvData := fmt.Sprintf("%s.%sData", srv, usrv)
	srvCollection := fmt.Sprintf("%s.%sCollection", srv, usrv)
	dvar := fmt.Sprintf("d%s", shortSrv)
	dstruct := fmt.Sprintf("db%s", usrv)
	drows := fmt.Sprintf("d%sRows", shortSrv)
	jsapiLinks := fmt.Sprintf("%s.%s", "jsonapi", "Links")
	jsapiMeta := fmt.Sprintf("%s.%s", "jsonapi", "Meta")
	jsapiPagination := fmt.Sprintf("%s.%s", "jsonapi", "Pagination")
	rcv := Id("s").Op("*").Id(usrv + "Service")
	intId := Id("id").Int64()
	aphgrpcPath := "github.com/dictyBase/apihelpers/aphgrpc"

	f.Const().Defs(
		Id(srv + "DbTable").Op("=").Lit("--TABLE NAME--"),
	)
	f.Var().Id(srv + "Cols").Op("=").Index().String().Values(Lit("List of column names ..."))
	f.Type().Id(dstruct).Struct()
	f.Type().Id(usrv + "Service").Struct(
		Op("*").Qual(aphgrpcPath, "Service"),
	)
	f.Comment("-- Constructor")
	f.Func().Id("New"+usrv+"Service").
		Params(
			Id("dbh").Op("*").Qual("gopkg.in/mgutz/dat.v1/sqlx-runner", "DB"),
			Id("pathPrefix").String(),
		).
		Params(
			Op("*").Id(usrv + "Service"),
		).
		Block(
			Return(
				Op("&").Id(usrv + "Service").
					Values(
						Op("&").Id("aphgrpc.Service").
							Values(
								Dict{
									Id("Resource"):     Id(srv + "s"),
									Id("Dbh"):          Id("dbh"),
									Id("PathPrefix"):   Id("pathPrefix"),
									Id("Include"):      Index().String().Values(Lit("--allowed includes---")),
									Id("FilToColumns"): Map(String()).String().Values(Lit("--field to column maps")),
									Id("ReqAttrs"):     Index().String().Values(Lit("-- required attributes ---")),
								},
							),
					),
			),
		)

	f.Comment("-- Functions that queries the storage and generates resource object")
	f.Func().
		Params(rcv).
		Id("existsResource").
		Params(intId).
		Params(Error()).
		Block(
			List(Id("_"), Err()).Op(":=").
				Id("s").Dot("Dbh").Dot("Select").
				Call(Lit("select columns---")).
				Dot("From").
				Call(Lit("--some table---")).
				Dot("Where").
				Call(Lit("--where clause---")).
				Dot("Exec").
				Call(),
			Return(Err()),
		)

	f.Func().Params(rcv).Id("getResourceWithSelectedAttr").
		Params(intId).
		Params(
			Op("*").Qual(srvPath, usrv),
			Error(),
		).
		Block(
			Id(dvar).Op(":=").Id(dstruct).Values(),
			Id("columns").
				Op(":=").
				Id("s").Dot("MapFieldsToColumns").
				Call(
					Id("s").
						Dot("Params").
						Dot("Fields"),
				),
			Err().Op(":=").
				Id("s").
				Dot("Dbh").
				Dot("Select").
				Call(Id("columns").Op("...")).
				Dot("From").
				Call(Lit("--some table--")).
				Dot("Where").
				Call(Lit("--where clause---")).
				Dot("QueryStruct").
				Call(Id(dvar)),
			If(
				Err().Op("!=").Nil(),
			).
				Block(
					Return(
						List(
							Op("&").Id(srvPayload).Values(),
							Err(),
						),
					),
				),
			Return(
				Id("s").
					Dot("buildResource").
					Call(
						Id("id"),
						Id("s").
							Dot("dbToResourceAttributes").
							Call(Id(dstruct)),
					),
				Nil(),
			),
		)

	f.Func().Params(rcv).Id("getResource").
		Params(intId).
		Params(Op("*").Id(srvPayload), Error()).
		Block(
			Id(dvar).Op(":=").Op("&").Id(dstruct).Values(),
			Err().Op(":=").Id("s").Dot("Dbh").
				Dot("Select").Call().
				Dot("From").Call().
				Dot("Where").Call().
				Dot("QueryStruct").
				Call(Id(dstruct)),
			Return(
				Id("s").Dot("buildResource").Call(
					Id("id"),
					Id("s").Dot("dbToResourceAttributes").Call(Id(dvar)),
				),
				Nil(),
			),
		)

	f.Comment("-- Functions that queries the storage and generates database mapped objects")
	f.Func().Params(rcv).Id("getAllRows").
		Params().
		Params(
			Index().Op("*").Id(dstruct),
			Error(),
		).Block(
		Var().Id(drows).Index().Op("*").Id(dstruct),
		Err().Op(":=").Id("s").Dot("Dbh").
			Dot("Select").Call().
			Dot("From").Call().
			Dot("QueryStruct").
			Call(Id(drows)),
		Return(Id(drows), Err()),
	)

	f.Comment("-- Functions that builds up the various parts of the final user resource objects")
	f.Func().Params(rcv).Id("buildResourceData").
		Params(
			intId,
			Id("attr").Op("*").Id(srvAttr),
		).Params(
		Op("*").Id(srvData),
	).Block(
		Return(
			Op("&").Id(srvData).Values(
				Dict{
					Id("Type"):       Id("s").Dot("GetResourceName").Call(),
					Id("Id"):         Id("id"),
					Id("Attributes"): Id("attr"),
					Id("Links"): Op("&").Id(jsapiLinks).Values(
						Dict{
							Id("Self"): Id("s").Dot("GenResourceSelfLink").Call(Id("id")),
						}),
					Id("Relationships"): Op("&").Id("SomeRelationships").Values(),
				},
			),
		),
	)

	f.Func().Params(rcv).Id("buildResource").
		Params(
			intId,
			Id("attr").Op("*").Id(srvAttr),
		).Params(
		Op("*").Id(srvPayload),
	).Block(
		Return(
			Op("&").Id(srvPayload).Values(
				Dict{
					Id("Data"): Id("s").Dot("buildResourceData").Call(Id("id"), Id("attr")),
				}),
		),
	)

	f.Func().Params(rcv).Id("buildResourceRelationships").
		Params(
			intId,
			Id(srv).Op("*").
				Id(srvPayload),
		).Params(Error()).
		Block(
			Var().Id("allInc").
				Index().Op("*").
				Qual("github.com/golang/protobuf/ptypes/any", "Any"),
			Id(srv).Dot("Included").Op("=").Id("allInc"),
		)

	f.Comment("Functions that generates resource objects or parts of it from database mapped objects")

	f.Func().Params(rcv).Id("dbToResourceAttributes").
		Params(
			Id(dvar).Op("*").Id(dstruct),
		).
		Params(
			Op("*").Id(srvAttr),
		).
		Block(
			Return(
				Op("&").Id(srvAttr).Values(),
			),
		)

	f.Func().Params(rcv).Id("dbToCollResourceData").
		Params(
			Id(drows).Index().Op("*").Id(dstruct),
		).
		Params(
			Index().Op("*").Id(srvData),
		).
		Block(
			Var().Id("dslice").Index().Op("*").Id(srvData),
			For(
				List(Id("_"), Id("d")).
					Op(":=").Range().Id(drows),
			).
				Block(
					Id("dslice").Op("=").Append(
						Id("dslice"),
						Id("s").Dot("buildResourceData").Call(
							Id("d").Dot("something"),
							Id("s").Dot("dbToResourceAttributes").Call(Id("d")),
						),
					),
				),
			Return(Id("dslice")),
		)

	f.Func().Params(rcv).Id("dbToCollResource").
		Params(
			Id(drows).Index().Op("*").Id(dstruct),
		).
		Params(
			Op("*").Id(srvCollection),
		).
		Block(
			Return(
				Op("&").Id(srvCollection).Values(
					Dict{
						Id("Data"): Id("s").Dot("dbToCollResourceData").Call(Id(drows)),
						Id("Links"): Op("&").Id(jsapiLinks).Values(
							Dict{
								Id("Self"): Id("s").Dot("GenCollResourceSelfLink").Call(),
							},
						),
					},
				),
			),
		)

	f.Func().Params(rcv).Id("dbToCollResourceWithPagination").
		Params(
			Id("count").Int64(),
			Id(drows).Index().Op("*").Id(dstruct),
			Id("pagenum").Int64(),
			Id("pagesize").Int64(),
		).
		Params(
			Op("*").Id(srvCollection),
		).
		Block(
			Id("dslice").Op(":=").Id("s").Dot("dbToCollResourceData").Call(Id(drows)),
			List(Id("jsLinks"), Id("pages")).
				Op(":=").
				Id("s").Dot("GetPagination").
				Call(
					Id("count"),
					Id("pagenum"),
					Id("pagesize"),
				),
			Return(
				Op("&").Id(srvCollection).
					Values(
						Dict{
							Id("Data"):  Id("dslice"),
							Id("Links"): Id("jsLinks"),
							Id("Meta"): Op("&").Id(jsapiMeta).Values(
								Dict{
									Id("Pagination"): Op("&").Id(jsapiPagination).Values(
										Dict{
											Id("Records"): Id("count"),
											Id("Total"):   Id("pages"),
											Id("Size"):    Id("pagesize"),
											Id("Number"):  Id("pagenum"),
										},
									),
								},
							),
						},
					),
			),
		)

	f.Func().Params(rcv).Id("dbToCollResourceWithRelAndPagination").
		Params(
			Id("count").Int64(),
			Id(drows).Index().Op("*").Id(dstruct),
			Id("pagenum").Int64(),
			Id("pagesize").Int64(),
		).
		Params(
			Op("*").Id(srvCollection),
			Error(),
		).
		Block(
			Id("dslice").Op(":=").Id("s").Dot("dbToCollResourceData").Call(Id(drows)),
			List(Id("jsLinks"), Id("pages")).
				Op(":=").
				Id("s").Dot("GetPagination").
				Call(
					Id("count"),
					Id("pagenum"),
					Id("pagesize"),
				),
			Return(
				Op("&").Id(srvCollection).
					Values(
						Dict{
							Id("Data"):     Id("dslice"),
							Id("Links"):    Id("jsLinks"),
							Id("Included"): Id("relations-object"),
							Id("Meta"): Op("&").Id(jsapiMeta).Values(
								Dict{
									Id("Pagination"): Op("&").Id(jsapiPagination).Values(
										Dict{
											Id("Records"): Id("count"),
											Id("Total"):   Id("pages"),
											Id("Size"):    Id("pagesize"),
											Id("Number"):  Id("pagenum"),
										},
									),
								},
							),
						},
					),
			),
		)

	f.Func().Params(rcv).Id("attrTo" + dstruct).
		Params(
			Id("attr").Op("*").Id(srvAttr),
		).
		Params(
			Op("*").Id(dstruct),
		).
		Block(
			Return(
				Op("&").Id(dstruct).Values(),
			),
		)

	f.Func().Params(rcv).Id("convertAllToAny").
		Params(Id(srv+"s").Index().Op("*").Id(srvData)).
		Params(Index().Op("*").Id("any.Any"), Error()).
		Block(
			Id("aslice").Op(":=").
				Make(
					Op("*").Index().Id("any.Any"),
					Len(Id(srv+"s")),
				),
			For(
				List(Id("i"), Id("u")).
					Op(":=").Range().Id(srv+"s"),
			).Block(
				List(Id("pkg"), Err()).
					Op(":=").
					Id("ptypes").
					Dot("MarshalAny").Call(Id("u")),
				If(
					Err().Op("!=").Nil(),
				).Block(
					Return(List(Id("aslice"), Err())),
				),
				Id("aslice").Index(Id("i")).Op("=").Id("pkg"),
			),
			Return(List(Id("aslice"), Nil())),
		)

	if err := f.Render(output); err != nil {
		return cli.NewExitError("unable to render output", 2)
	}
	return nil
}
