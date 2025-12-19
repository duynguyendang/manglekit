# Math Solver Example

This example demonstrates how to use Manglekit's "Dual Brain" architecture to solve a math problem ($2^t = t^2$).

## Architecture

1.  **System 1 (Gemini)**: The `gemini_solver` action uses an LLM to propose potential candidate solutions efficiently.
2.  **System 2 (Logic Engine)**: The `protocol.dl` policy defines the governance flow, routing the proposed candidates to a rigorous verification step.
3.  **Tool (Verifier)**: The `verifier` action uses exact mathematical computation (Go code) to validate the candidates.

## How to Run

1.  Set your `GOOGLE_API_KEY`:
    
    ```bash
    export GOOGLE_API_KEY="your-key-here"
    ```

2.  Run the example:

    ```bash
    go run ./examples/math_solver
    ```

## Expected Flow

1.  **Input**: "Find t when 2^t=t^2"
2.  **Gemini**: Proposes `[2, 4, -0.7667]`
3.  **Logic Engine**: Sees `status("proposed")` -> Routes to `verifier`.
4.  **Verifier**: Checks $2^t$ vs $t^2$. Confirms 2 and 4. Rejects or Approves -0.7667 based on precision.
5.  **Logic Engine**: Sees `status("verified")` -> Routes to `printer`.
6.  **Output**: Displays valid solutions.
