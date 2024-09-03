package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"gonum.org/v1/gonum/mat"
)

// Model represents the state of the application
type model struct {
	matrixA mat.Matrix // Stores the first user-entered matrix or vector
	matrixB mat.Matrix // Stores the second user-entered matrix or vector, if needed
	state   string     // Current state of the application
	input   string     // Current user input
	err     error      // Stores any errors that occur
}

// initialModel sets up the initial state of the application.
func initialModel() model {
	return model{
		state: "inputA", // Start in the input state for matrix A
	}
}

// Init is called when the program starts
func (m model) Init() tea.Cmd {
	return nil
}

// Update is called when an event occurs, such as a key press
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q": // Quit the application
			return m, tea.Quit
		case "enter":
			if m.state == "inputA" {
				// Parse input into the first matrix
				m.matrixA, m.err = parseMatrix(m.input)
				if m.err != nil {
					m.state = "error" // Switch to the error state if parsing fails
				} else {
					m.state = "select" // Switch to the select state if parsing succeeds
					m.input = ""       // Clear input after parsing
				}
			} else if m.state == "inputB" {
				// Parse input into the second matrix
				m.matrixB, m.err = parseMatrix(m.input)
				if m.err != nil {
					m.state = "error" // Switch to the error state if parsing fails
				} else {
					m.state = "selectOp" // Switch to the operation selection state
					m.input = ""         // Clear input after parsing
				}
			} else if m.state == "select" {
				// Based on user input, proceed to perform an operation or request the second matrix
				switch m.input {
				case "det":
					if isVector(m.matrixA) {
						m.input = "Determinant is only defined for matrices."
					} else {
						det := mat.Det(m.matrixA)
						m.input = fmt.Sprintf("Determinant: %v", det)
					}
					m.state = "result"
				case "norm":
					norm := mat.Norm(m.matrixA, 2)
					m.input = fmt.Sprintf("Norm: %v", norm)
					m.state = "result"
				case "nullspace":
					nullspace, err := calculateNullspace(m.matrixA)
					if err != nil {
						m.input = fmt.Sprintf("Error calculating nullspace: %v", err)
					} else {
						m.input = fmt.Sprintf("Nullspace:\n%v", matrixToString(nullspace))
					}
					m.state = "result"
				case "inner", "outer", "multiply":
					// Request the second matrix or vector for further operations
					m.input = ""
					m.state = "inputB"
				default:
					m.input = "Invalid selection. Please choose 'det', 'norm', 'nullspace', 'inner', 'outer', or 'multiply'.\n"
					m.state = "select"
				}
			} else if m.state == "selectOp" {
				// Perform operations based on the input matrices/vectors
				switch m.input {
				case "inner":
					if isVector(m.matrixA) && isVector(m.matrixB) {
						inner, _ := innerProduct(m.matrixA.(*mat.Dense), m.matrixB.(*mat.Dense))
						m.input = fmt.Sprintf("Inner Product: %v", inner)
					} else {
						m.input = "Inner product is only defined for vectors."
					}
					m.state = "result"
				case "outer":
					if isVector(m.matrixA) && isVector(m.matrixB) {
						outer := outerProduct(m.matrixA.(*mat.Dense), m.matrixB.(*mat.Dense))
						m.input = fmt.Sprintf("Outer Product:\n%v", matrixToString(outer))
					} else {
						m.input = "Outer product is only defined for vectors."
					}
					m.state = "result"
				case "multiply":
					if !isVector(m.matrixA) && !isVector(m.matrixB) {
						rA, cA := m.matrixA.Dims()
						rB, cB := m.matrixB.Dims()
						if cA != rB {
							m.input = "Matrices are not compatible for multiplication."
						} else {
							product := mat.NewDense(rA, cB, nil)
							product.Mul(m.matrixA, m.matrixB)
							m.input = fmt.Sprintf("Matrix Product:\n%v", matrixToString(product))
						}
					} else {
						m.input = "Matrix multiplication is only defined for matrices."
					}
					m.state = "result"
				default:
					m.input = "Invalid selection for operation. Please choose 'inner', 'outer', or 'multiply'."
					m.state = "selectOp"
				}
			} else if m.state == "result" || m.state == "error" {
				m.state = "inputA" // Reset to input state for matrix A
				m.input = ""       // Clear input
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1] // Remove the last character from input
			}
		default:
			m.input += msg.String() // Append typed characters to the input
		}
	}

	return m, nil
}

// View renders the user interface based on the current state
func (m model) View() string {
	switch m.state {
	case "inputA":
		return "Enter a matrix or vector (comma-separated values, semicolon-separated rows):\n" + m.input + "\n"
	case "inputB":
		return "Enter a second matrix or vector (comma-separated values, semicolon-separated rows):\n"
	case "select":
		matrixView := matrixToString(m.matrixA)
		return fmt.Sprintf("Matrix A:\n%s\nChoose an operation: det (Determinant), norm (Norm), nullspace (Nullspace), inner (Inner Product), outer (Outer Product), multiply (Matrix Multiplication)\n%s", matrixView, m.input)
	case "selectOp":
		matrixViewA := matrixToString(m.matrixA)
		matrixViewB := matrixToString(m.matrixB)
		return fmt.Sprintf("Matrix A:\n%s\nMatrix B:\n%s\nChoose an operation: inner (Inner Product), outer (Outer Product), multiply (Matrix Multiplication)\n%s", matrixViewA, matrixViewB, m.input)
	case "result":
		return m.input + "\n\nPress enter to continue..."
	case "error":
		return "Error: " + m.err.Error() + "\n\nPress enter to try again..."
	default:
		return "Unknown state"
	}
}

// matrixToString converts a matrix to a string for displaying it in the CLI
func matrixToString(matrix mat.Matrix) string {
	var sb strings.Builder
	fa := mat.Formatted(matrix, mat.Prefix(""), mat.Squeeze())
	fmt.Fprintf(&sb, "%v", fa)
	return sb.String()
}

// parseMatrix parses the user input into a matrix using the gonum package
func parseMatrix(input string) (mat.Matrix, error) {
	rows := strings.Split(input, ";")
	data := []float64{}
	rowLen := -1
	for _, row := range rows {
		cols := strings.Split(strings.TrimSpace(row), ",")
		if rowLen == -1 {
			rowLen = len(cols)
		} else if rowLen != len(cols) {
			return nil, fmt.Errorf("rows have inconsistent lengths")
		}
		for _, col := range cols {
			var value float64
			_, err := fmt.Sscanf(col, "%f", &value)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %v", col)
			}
			data = append(data, value)
		}
	}
	matrix := mat.NewDense(len(rows), rowLen, data)
	return matrix, nil
}

// isVector checks if the provided matrix is a vector (either a row or a column vector)
func isVector(matrix mat.Matrix) bool {
	r, c := matrix.Dims()
	return r == 1 || c == 1
}

// calculateNullspace computes the nullspace of a matrix using SVD
func calculateNullspace(matrix mat.Matrix) (mat.Matrix, error) {
	var svd mat.SVD
	ok := svd.Factorize(matrix, mat.SVDThin)
	if !ok {
		return nil, fmt.Errorf("SVD factorization failed")
	}

	rows, cols := matrix.Dims()

	// Correctly initialize U matrix to store left singular vectors
	u := mat.NewDense(rows, rows, nil)
	svd.UTo(u)

	s := svd.Values(nil)

	// Identify columns in U corresponding to near-zero singular values
	var nullspaceCols []int
	for i, singularValue := range s {
		if singularValue <= 1e-12 {
			nullspaceCols = append(nullspaceCols, i)
		}
	}

	// If there are no columns corresponding to near-zero singular values, return an empty nullspace
	if len(nullspaceCols) == 0 {
		return mat.NewDense(cols, 0, nil), nil // Empty nullspace
	}

	// Construct the nullspace matrix
	nullspace := mat.NewDense(rows, len(nullspaceCols), nil)
	for j, colIndex := range nullspaceCols {
		for i := 0; i < rows; i++ {
			nullspace.Set(i, j, u.At(i, colIndex))
		}
	}

	return nullspace, nil
}

// outerProduct computes the outer product of two vectors
func outerProduct(a, b *mat.Dense) *mat.Dense {
	rA, cA := a.Dims()
	rB, cB := b.Dims()

	if cA != 1 && rA != 1 {
		panic("First input is not a vector")
	}

	if cB != 1 && rB != 1 {
		panic("Second input is not a vector")
	}

	outer := mat.NewDense(rA*rB, cA*cB, nil)
	outer.Mul(a, b.T())
	return outer
}

// innerProduct computers the inner product of two vectors
func innerProduct(a, b *mat.Dense) (float64, error) {
	rA, cA := a.Dims()
	rB, cB := b.Dims()

	// Check if both inputs are vectors
	if !(isVector(a) && isVector(b)) {
		return 0, fmt.Errorf("Inner product is only defined for vectors.")
	}

	// Convert to 1D slices and compute dot product
	if rA == 1 && cB == 1 {
		// a is a row vector and b is a column vector
		return mat.Dot(a.RowView(0), b.ColView(0)), nil
	} else if cA == 1 && rB == 1 {
		// a is a column vector and b is a row vector
		return mat.Dot(a.ColView(0), b.RowView(0)), nil
	} else if rA == 1 && rB == 1 && cA == cB {
		// Both a and b are row vectors of the same length
		return mat.Dot(a.RowView(0), b.RowView(0)), nil
	} else if cA == 1 && cB == 1 && rA == rB {
		// Both a and b are column vectors of the same length
		return mat.Dot(a.ColView(0), b.ColView(0)), nil
	} else {
		return 0, fmt.Errorf("Vectors must have the same dimension.")
	}
}

// main is the entry point of the application
func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
