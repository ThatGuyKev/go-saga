package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

type StepPhase string

const (
	STEP_FORWARD  StepPhase = "STEP_FORWARD"
	STEP_BACKWARD StepPhase = "STEP_BACKWARD"
)

type Command struct {
	command func(P interface{}) error
}

type Saga struct {
	index int
	phase StepPhase
}
type SagaMessage struct {
	payload interface{}
	saga    Saga
}
type SagaDefinition struct {
	channelName string
	phases      map[StepPhase]Command
}

type SagaDefinitionBuilder struct {
	index           int
	sagaDefinitions []SagaDefinition
	sagaChan        chan SagaMessage
}

type ISagaDefinitionBuilder interface {
	step(channel string, stepForward Command, stepBackward Command)
	start(payload interface{})
	listen()
}

func (s *SagaDefinitionBuilder) checkIndex() error {
	if s.index < 0 {
		return errors.New("invalid index")
	}
	return nil
}

func (s *SagaDefinitionBuilder) step(channelName string, stepForward Command, stepBackward Command) {
	s.index += 1
	phases := make(map[StepPhase]Command)
	phases[STEP_FORWARD] = stepForward
	phases[STEP_BACKWARD] = stepBackward
	s.sagaDefinitions = append(s.sagaDefinitions, SagaDefinition{channelName: channelName, phases: phases})
}

func (s *SagaDefinitionBuilder) makeStepForward(index int, payload interface{}) {

	if index >= len(s.sagaDefinitions) {
		fmt.Println("Error baby :)")
		return
	}
	fmt.Println(payload.(User).name, "is making Step Forward to ", s.sagaDefinitions[index].channelName)

	msg := SagaMessage{payload: payload, saga: Saga{index: index, phase: StepPhase(STEP_FORWARD)}}
	// fmt.Println("pushing", msg, "to channel")
	s.sagaChan <- msg

}

func (s *SagaDefinitionBuilder) makeStepBackward(index int, payload interface{}) {
	if index < 0 {
		fmt.Println("Error baby :)")
		return
	}
	fmt.Println(payload.(User).name, "is rolling back to ", s.sagaDefinitions[index].channelName)

	msg := SagaMessage{payload: payload, saga: Saga{index: index, phase: StepPhase(STEP_BACKWARD)}}

	s.sagaChan <- msg
}

func (s *SagaDefinitionBuilder) start(payload interface{}) {
	s.makeStepForward(0, payload)
}

func NewSagaDefinition() ISagaDefinitionBuilder {
	return &SagaDefinitionBuilder{
		sagaChan: make(chan SagaMessage),
	}

}

func (s *SagaDefinitionBuilder) listen() {
	for {
		sm := <-s.sagaChan
		switch sm.saga.phase {
		case StepPhase(STEP_FORWARD):
			stepForward := s.sagaDefinitions[sm.saga.index].phases[(STEP_FORWARD)].command
			err := stepForward(sm.payload)
			if err != nil || (sm.saga.index > 5 && sm.payload.(User).name == "ming") {
				go s.makeStepBackward(sm.saga.index-1, sm.payload)
			} else {
				go s.makeStepForward(sm.saga.index+1, sm.payload)

			}
		case StepPhase(STEP_BACKWARD):
			stepBackward := s.sagaDefinitions[sm.saga.index].phases[StepPhase(STEP_BACKWARD)].command
			stepBackward(sm.payload)
			go s.makeStepBackward(sm.saga.index-1, sm.payload)
		}
	}
}

type User struct {
	id   string
	name string
}

func doSomething(P interface{}) error {
	if _, ok := P.(User); ok {
		time.Sleep(time.Second)
		// fmt.Println("I'm doing something for ", user.name, " :)")
		return nil

	} else {
		return errors.New("Mismatch interface")
	}
}

func compensation(P interface{}) error {
	time.Sleep(time.Second)

	return nil
}

func main() {

	e := echo.New()
	sagaProcessor := NewSagaDefinition()
	sagaProcessor.step("START", Command{doSomething}, Command{compensation})
	sagaProcessor.step("CREATE_EMPTY_CART", Command{doSomething}, Command{compensation})
	sagaProcessor.step("ADD_ITEM_REQUEST", Command{doSomething}, Command{compensation})
	sagaProcessor.step("RETRIEVE_PRODUCTS", Command{doSomething}, Command{compensation})
	sagaProcessor.step("VALIDATE_CART", Command{doSomething}, Command{compensation})
	sagaProcessor.step("MERGE_SIMILAR_ITEMS", Command{doSomething}, Command{compensation})
	sagaProcessor.step("PRICE_CART", Command{doSomething}, Command{compensation})
	sagaProcessor.step("CALCULATE_TOTALS", Command{doSomething}, Command{compensation})
	sagaProcessor.step("PERSIST_NEW_CART", Command{doSomething}, Command{compensation})
	sagaProcessor.step("END", Command{doSomething}, Command{compensation})
	go func() {
		sagaProcessor.listen()
	}()
	e.GET("/:name", func(c echo.Context) error {
		user := User{id: "id", name: c.Param("name")}

		sagaProcessor.start(user)

		return nil

	})

	e.Logger.Fatal(e.Start(":7001"))

}
