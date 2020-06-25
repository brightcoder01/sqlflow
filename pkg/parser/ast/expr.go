// Copyright 2020 The SQLFlow Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ast

import (
	"fmt"
	"strings"
)

// Expr defines an expression.
type Expr struct {
	// The design inherits from Lisp's S-expression.  It represents a
	// literal if typ is zero.  The type of the literal is identified by
	// typ, in particular, NUMBER, IDENT, or STRING, defined by typ and val,
	// and the printing form is in val; otherwise, typ and val are
	// ignorable, but sexp it is a Lisp S-expression and funcall indicate if
	// sexp is a function call or usual unary or binary expressions.
	// funcall is useful for printing the expression.  We should not
	// required funcall because if sexp is a function call, sexp[0].typ must
	// be IDENT.  However, goyacc cannot allow us to use externally defined
	// constant symbol IDs in *.y files.  And the *.y file needs to import
	// package ast, so we cannot have ast to import package parser so to use
	// symbol IDs generated by goyacc.  This constaint forces us to add this
	// ad-hoc field funcall.
	typ     int
	val     string
	sexp    ExprList
	funcall bool
}

// ExprList represent a list of expressions.  For example, the parser generates
// an ExprList from parameters of a function call.
type ExprList []*Expr

// IsLiteral returns true if e is a literal value; otherwise, we say that e is a
// compound expression.
func (e Expr) IsLiteral() bool {
	return e.typ != 0
}

// IsFuncall returns true if the expression represents a function call.
func (e Expr) IsFuncall() bool {
	return !e.IsLiteral() && e.funcall
}

// IsVariadic returns true if the expression is a list "[...]".
func (e Expr) IsVariadic() bool {
	return !e.IsLiteral() && !e.IsFuncall() && len(e.sexp) > 0 &&
		(e.sexp[0].typ == '[' || e.sexp[0].typ == '(')
}

// IsUnary returns true if e is a binary expression.
func (e Expr) IsUnary() bool {
	return !e.IsLiteral() && !e.IsFuncall() && !e.IsVariadic() &&
		len(e.sexp) == 2
}

// IsBinary returns true if e is a binary expression.
func (e Expr) IsBinary() bool {
	return !e.IsLiteral() && !e.IsFuncall() && !e.IsVariadic() &&
		len(e.sexp) == 3
}

// NewLiteral returns a literal expression.
func NewLiteral(typ int, val string) (*Expr, error) {
	if typ == 0 {
		return nil, fmt.Errorf("typ 0 is hold as special and cannot be used")
	}
	return &Expr{
		typ: typ,
		val: val,
	}, nil
}

// NewUnary returns a unary exprression.
func NewUnary(typ int, op string, od1 *Expr) (*Expr, error) {
	oprt, e := NewLiteral(typ, op)
	if e != nil {
		return nil, e
	}
	if od1 == nil {
		return nil, fmt.Errorf("The operand of a unary expression is nil")
	}
	return &Expr{
		sexp: append(ExprList{oprt}, od1),
	}, nil
}

// NewBinary returns a binary expression.
func NewBinary(typ int, op string, od1 *Expr, od2 *Expr) (*Expr, error) {
	oprt, e := NewLiteral(typ, op)
	if e != nil {
		return nil, e
	}
	if od1 == nil {
		return nil, fmt.Errorf("The left operand of a binary expression is nil")
	}
	if od2 == nil {
		return nil, fmt.Errorf("The right operand of a binary expression is nil")
	}
	return &Expr{
		sexp: append(ExprList{oprt}, od1, od2),
	}, nil
}

// NewVariadic returns a variadic expression.
func NewVariadic(typ int, op string, ods ExprList) (*Expr, error) {
	if typ != '[' && typ != '(' {
		return nil, fmt.Errorf("Only [ and ( are supported with variadic expression")
	}
	if op != fmt.Sprintf("%c", typ) {
		return nil, fmt.Errorf("Given typ %c, op must be \"%c\", got \"%s\"", typ, typ, op)
	}
	oprt, e := NewLiteral(typ, op)
	if e != nil {
		return nil, e
	}
	return &Expr{
		sexp: append(ExprList{oprt}, ods...),
	}, nil
}

// NewFuncall returns an expression representing a function call.
func NewFuncall(typ int, op string, oprd ExprList) (*Expr, error) {
	fn, e := NewLiteral(typ, op)
	if e != nil {
		return nil, e
	}
	return &Expr{
		sexp:    append(ExprList{fn}, oprd...),
		funcall: true,
	}, nil
}

func (e *Expr) String() string {
	if !e.IsLiteral() { /* a compound expression */
		switch {
		case e.IsBinary():
			return fmt.Sprintf("%s %s %s", e.sexp[1], e.sexp[0].val, e.sexp[2])
		case e.IsUnary():
			return fmt.Sprintf("%s %s", e.sexp[0], e.sexp[1])
		case e.IsVariadic():
			switch t := e.sexp[0].typ; t {
			case '[':
				return "[" + strings.Join(e.cdr(), ", ") + "]"
			case '(':
				return "(" + strings.Join(e.cdr(), ", ") + ")"
			default:
				panic(fmt.Errorf("Stringize compound expression with unknown type %d", t))
			}
		case e.IsFuncall(): /* function call */
			return e.sexp[0].val + "(" + strings.Join(e.cdr(), ", ") + ")"
		}
	}
	return fmt.Sprintf("%s", e.val)
}

/* Like Lisp's builtin function cdr. */
func (e *Expr) cdr() (r []string) {
	for i := 1; i < len(e.sexp); i++ {
		r = append(r, e.sexp[i].String())
	}
	return r
}
