package qdrmanagement

import (
	"encoding/json"
	entities "github.com/gaohoward/shipshape/pkg/apps/qdrouterd/qdrmanagement/entities"
	"github.com/gaohoward/shipshape/pkg/framework"
	"reflect"
	"time"
)

const (
	timeout time.Duration = 60 * time.Second
)

var (
	queryCommand = []string{"qdmanage", "query", "--type"}
)

// QdmanageQuery executes a "qdmanager query" command on the provided pod, returning
// a slice of entities of the provided "entity" type.
func QdmanageQuery(c framework.ContextData, pod string, entity entities.Entity, fn func(entities.Entity) bool) ([]entities.Entity, error) {
	// Preparing command to execute
	command := append(queryCommand, entity.GetEntityId())
	kubeExec := framework.NewKubectlExecCommand(c, pod, timeout, command...)
	jsonString, err := kubeExec.Exec()
	if err != nil {
		return nil, err
	}

	// Using reflection to get a slice instance of the concrete type
	vo := reflect.TypeOf(entity)
	v := reflect.SliceOf(vo)
	nv := reflect.New(v)
	//fmt.Printf("v    - %T - %v\n", v, v)
	//fmt.Printf("nv   - %T - %v\n", nv, nv)

	// Unmarshalling to a slice of the concrete Entity type provided via "entity" instance
	err = json.Unmarshal([]byte(jsonString), nv.Interface())
	if err != nil {
		//fmt.Printf("ERROR: %v\n", err)
		return nil, err
	}

	// Adding each parsed concrete Entity to the parsedEntities
	parsedEntities := []entities.Entity{}
	for i := 0; i < nv.Elem().Len(); i++ {
		candidate := nv.Elem().Index(i).Interface().(entities.Entity)

		// If no filter function provided, just add
		if fn == nil {
			parsedEntities = append(parsedEntities, candidate)
			continue
		}

		// Otherwhise invoke to determine whether to include
		if fn(candidate) {
			parsedEntities = append(parsedEntities, candidate)
		}
	}

	return parsedEntities, err
}

// QdmanageQueryWithRetries calls QdmanageQuery based on given delay and timeout, till the
// done function returns true (or if done function is nil).
func QdmanageQueryWithRetries(c framework.ContextData, pod string, entity entities.Entity,
	delaySecs int, timeoutSecs int, filter func(entities.Entity) bool,
	done func(es []entities.Entity, err error) bool) (es []entities.Entity, err error) {

	// Wait timeout
	timeout := time.Duration(timeoutSecs) * time.Second

	// Channel to notify result or timeout
	for t := time.Now(); time.Since(t) < timeout; time.Sleep(time.Duration(delaySecs) * time.Second) {
		es, err = QdmanageQuery(c, pod, entity, filter)
		if done == nil || done(es, err) {
			return
		}
	}

	return
}
