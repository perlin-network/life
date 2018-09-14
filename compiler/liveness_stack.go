// Stack datastructure
//
// Initial implementation by https://github.com/golang-collections/collections,
// MIT see licence https://github.com/golang-collections/collections/blob/master/LICENSE
package compiler

type (
	livenessTraversalStack struct {
		top    *node
		length int
	}
	node struct {
		value *livenessBasicBlock
		prev  *node
	}
)

// Create a new livenessTraversalStack
func NewLivenessTraversalStack() *livenessTraversalStack {
	return &livenessTraversalStack{nil, 0}
}

// Return the number of items in the stack
func (this *livenessTraversalStack) Len() int {
	return this.length
}

// View the top item on the stack
func (this *livenessTraversalStack) Peek() *livenessBasicBlock {
	if this.length == 0 {
		return nil
	}
	return this.top.value
}

// Pop the top item of the stack and return it
func (this *livenessTraversalStack) Pop() *livenessBasicBlock {
	if this.length == 0 {
		return nil
	}

	n := this.top
	this.top = n.prev
	this.length--
	return n.value
}

// Push a value onto the top of the stack
func (this *livenessTraversalStack) Push(value *livenessBasicBlock) {
	n := &node{value, this.top}
	this.top = n
	this.length++
}
