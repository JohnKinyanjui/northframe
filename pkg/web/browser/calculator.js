const operatorSymbols = {
  add: "+",
  subtract: "−",
  multiply: "×",
  divide: "÷",
};

function preciseNumber(value) {
  if (!Number.isFinite(value)) return null;
  return Number.parseFloat(value.toPrecision(12));
}

function calculate(left, right, operator) {
  if (operator === "add") return preciseNumber(left + right);
  if (operator === "subtract") return preciseNumber(left - right);
  if (operator === "multiply") return preciseNumber(left * right);
  if (operator === "divide") return right === 0 ? null : preciseNumber(left / right);
  return right;
}

function mountCalculator(root) {
  const displayNode = root.querySelector("[data-nf-display]");
  const expressionNode = root.querySelector("[data-nf-expression]");
  const clearNode = root.querySelector('[data-nf-key="clear"]');

  let current = displayNode.textContent.trim() || "0";
  let stored = null;
  let operator = null;
  let waitingForOperand = false;

  function render() {
    displayNode.textContent = current;
    clearNode.textContent = current === "0" && stored === null ? "AC" : "C";
  }

  function resetAfterError() {
    if (current !== "Error") return false;
    current = "0";
    stored = null;
    operator = null;
    waitingForOperand = false;
    expressionNode.textContent = "Ready";
    return true;
  }

  function inputDigit(digit) {
    resetAfterError();
    if (waitingForOperand || current === "0") {
      current = digit;
      waitingForOperand = false;
    } else if (current.replace("-", "").replace(".", "").length < 12) {
      current += digit;
    }
    render();
  }

  function inputDecimal() {
    resetAfterError();
    if (waitingForOperand) {
      current = "0.";
      waitingForOperand = false;
    } else if (!current.includes(".")) {
      current += ".";
    }
    render();
  }

  function setError(message) {
    current = "Error";
    stored = null;
    operator = null;
    waitingForOperand = true;
    expressionNode.textContent = message;
    render();
  }

  function chooseOperator(nextOperator) {
    if (resetAfterError()) render();
    const input = Number.parseFloat(current);

    if (operator && !waitingForOperand) {
      const result = calculate(stored, input, operator);
      if (result === null) {
        setError("Cannot divide by zero");
        return;
      }
      current = String(result);
      stored = result;
    } else {
      stored = input;
    }

    operator = nextOperator;
    waitingForOperand = true;
    expressionNode.textContent = `${current} ${operatorSymbols[operator]}`;
    render();
  }

  function equals() {
    if (!operator || stored === null || current === "Error") return;
    const right = Number.parseFloat(current);
    const left = stored;
    const activeOperator = operator;
    const result = calculate(left, right, activeOperator);
    if (result === null) {
      setError("Cannot divide by zero");
      return;
    }
    expressionNode.textContent = `${left} ${operatorSymbols[activeOperator]} ${right} =`;
    current = String(result);
    stored = null;
    operator = null;
    waitingForOperand = true;
    render();
  }

  function clear() {
    current = "0";
    stored = null;
    operator = null;
    waitingForOperand = false;
    expressionNode.textContent = "Ready";
    render();
  }

  function backspace() {
    if (resetAfterError() || waitingForOperand) {
      render();
      return;
    }
    current = current.length > 1 ? current.slice(0, -1) : "0";
    if (current === "-") current = "0";
    render();
  }

  function toggleSign() {
    if (resetAfterError() || current === "0") {
      render();
      return;
    }
    current = current.startsWith("-") ? current.slice(1) : `-${current}`;
    render();
  }

  function percent() {
    if (resetAfterError()) {
      render();
      return;
    }
    current = String(preciseNumber(Number.parseFloat(current) / 100));
    waitingForOperand = false;
    render();
  }

  function activate(key) {
    if (/^\d$/.test(key)) return inputDigit(key);
    if (key === "decimal") return inputDecimal();
    if (key === "add" || key === "subtract" || key === "multiply" || key === "divide") return chooseOperator(key);
    if (key === "equals") return equals();
    if (key === "clear") return clear();
    if (key === "backspace") return backspace();
    if (key === "sign") return toggleSign();
    if (key === "percent") return percent();
  }

  for (const button of root.querySelectorAll("[data-nf-key]")) {
    button.addEventListener("click", () => {
      root.focus({ preventScroll: true });
      activate(button.getAttribute("data-nf-key"));
    });
  }

  root.addEventListener("keydown", (event) => {
    const keys = {
      ".": "decimal",
      ",": "decimal",
      "+": "add",
      "-": "subtract",
      "*": "multiply",
      "x": "multiply",
      "/": "divide",
      "%": "percent",
      Enter: "equals",
      "=": "equals",
      Escape: "clear",
      Backspace: "backspace",
    };
    const action = /^\d$/.test(event.key) ? event.key : keys[event.key];
    if (!action) return;
    event.preventDefault();
    activate(action);
  });

  render();
}

export function mountCalculators(root = document) {
  for (const calculator of root.querySelectorAll("[data-nf-calculator]")) {
    mountCalculator(calculator);
  }
}
