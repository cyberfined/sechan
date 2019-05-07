package proto

import (
	"errors"
	"log"
)

const CommandLength = 4

type Command [CommandLength]byte

type CommandParser struct {
	handlers    map[Command]interface{}
	typeChecker func(interface{}) bool
}

var (
	info = Command{'I', 'N', 'F', 'O'}
	list = Command{'L', 'I', 'S', 'T'}
	send = Command{'S', 'E', 'N', 'D'}
	file = Command{'F', 'I', 'L', 'E'}
	seek = Command{'S', 'E', 'E', 'K'}
	refo = Command{'R', 'E', 'F', 'O'}
	reli = Command{'R', 'E', 'L', 'I'}
	rese = Command{'R', 'E', 'S', 'E'}
	conn = Command{'C', 'O', 'N', 'N'}
	disc = Command{'D', 'I', 'S', 'C'}
	reer = Command{'R', 'E', 'E', 'R'}
	quit = Command{'Q', 'U', 'I', 'T'}

	ErrShortCommand = errors.New("command is too short")
)

func CreateCommandParser(typeChecker func(interface{}) bool) *CommandParser {
	return &CommandParser{
		handlers:    make(map[Command]interface{}),
		typeChecker: typeChecker,
	}
}

func (parser *CommandParser) AddCommand(cmd Command, handler interface{}) {
	if !parser.typeChecker(handler) {
		panic("wrong handler type")
	}

	parser.handlers[cmd] = handler
}

func (parser *CommandParser) GetHandler(data []byte) (interface{}, []byte, error) {
	var cmd Command

	if len(data) < CommandLength {
		return nil, nil, ErrShortCommand
	}

	copy(cmd[:], data)
	data = data[CommandLength:]

	handler, ok := parser.handlers[cmd]
	if !ok {
		return nil, nil, commandDoesntExists(cmd)
	}

	return handler, data, nil
}

func (parser *CommandParser) CommandLoop(rw PackageReadWriter, execute func(interface{}, []byte) error) {
	for rw != nil {
		buf, err := rw.ReadPackage()
		if err != nil {
			log.Println(err)
			return
		}

		handler, arg, err := parser.GetHandler(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		err = execute(handler, arg)
		if err != nil {
			log.Println(err)
			sendCommand(rw, reer, []byte(err.Error()))
		}
	}
}

func sendCommand(rw PackageReadWriter, cmd Command, body []byte) error {
	_, err := rw.WritePackage(packCommand(cmd, body))
	return err
}

func packCommand(cmd Command, body []byte) []byte {
	buf := make([]byte, CommandLength+len(body))
	copy(buf, cmd[:])
	copy(buf[CommandLength:], body)
	return buf
}

func commandDoesntExists(c Command) error {
	return errors.New("command " + string(c[:]) + " doesn't exists")
}

func wrongCommandData(c Command) error {
	return errors.New("wrong command data for " + string(c[:]))
}
