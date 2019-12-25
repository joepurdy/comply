package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/pkg/errors"
	"github.com/strongdm/comply/internal/config"
	"github.com/strongdm/comply/internal/model"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var procedureCommand = cli.Command{
	Name:      "procedure",
	ShortName: "proc",
	Usage:     "create ticket by procedure ID",
	ArgsUsage: "procedureID data.yaml",
	Action:    procedureAction,
	Before:    beforeAll(projectMustExist, ticketingMustBeConfigured),
}

func procedureAction(c *cli.Context) error {
	var (
		dataBytes []byte
		err       error
	)

	switch len(c.Args()) {
	case 0:
		return cli.NewExitError("provide a procedure ID", 1)
	case 1:
	case 2:
		dataBytes, err = ioutil.ReadFile(c.Args().Get(1))
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
	default:
		return cli.NewExitError("too many args provided", 1)
	}

	procedureID := c.Args().First()

	if err := renderProcedure(procedureID, dataBytes); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

func renderProcedure(id string, dataBytes []byte) error {
	procedures, err := model.ReadProcedures()
	if err != nil {
		return err
	}

	// unmarshal data
	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(dataBytes, &data)
	if err != nil {
		return err
	}

	ts, err := config.Config().TicketSystem()
	if err != nil {
		return errors.New("error in ticket system configuration")
	}

	tp := model.GetPlugin(model.TicketSystem(ts))

	for _, procedure := range procedures {
		if procedure.ID == id {
			// render body from data
			var w bytes.Buffer
			bodyTemplate, err := template.New("body").Parse(procedure.Body)
			if err != nil {
				w.WriteString(fmt.Sprintf("# Error processing template:\n\n%s\n", err.Error()))
			} else {
				bodyTemplate.Execute(&w, data)
			}
			renderedBody := w.String()

			err = tp.Create(&model.Ticket{
				Name: procedure.Name,
				Body: fmt.Sprintf("%s\n\n\n---\nProcedure-ID: %s", renderedBody, procedure.ID),
			}, []string{"comply", "comply-procedure"})
			if err != nil {
				return err
			}
			return nil
		}
	}

	return errors.New(fmt.Sprintf("unknown procedure ID: %s", id))
}
