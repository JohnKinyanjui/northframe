export function mountComponent(definition, root = document) {
  const bindings = (definition.bindings || []).flatMap((binding) =>
    Array.from(root.querySelectorAll(binding.selector)).map((element) => ({ ...binding, element })),
  );

  const render = () => {
    for (const binding of bindings) renderBinding(binding);
    root.dispatchEvent(new CustomEvent("northframe:update", { bubbles: true }));
  };

  for (const binding of bindings) {
    if (binding.kind !== "model" || typeof binding.write !== "function") continue;
    const eventName = binding.element.matches('input[type="checkbox"], input[type="radio"], select') ? "change" : "input";
    binding.element.addEventListener(eventName, () => {
      const value = readControl(binding.element);
      if (value === undefined) return;
      binding.write(value);
      render();
    });
  }

  for (const event of definition.events || []) {
    for (const element of root.querySelectorAll(event.selector)) {
      element.addEventListener(event.type, async (browserEvent) => {
        const result = event.run(browserEvent);
        render();
        if (result?.then) await result;
        render();
      });
    }
  }

  render();
  return { render };
}

export function readProps(root, marker) {
  const element = root.querySelector(`script[data-north-props="${marker}"]`);
  if (!element) return {};
  try {
    return JSON.parse(element.textContent || "{}");
  } catch (error) {
    throw new Error(`Northframe could not decode props for ${marker}`, { cause: error });
  }
}

function renderBinding(binding) {
  const value = binding.read();
  if (binding.kind === "text") binding.element.textContent = displayValue(value);
  if (binding.kind === "show") binding.element.hidden = !truthy(value);
  if (binding.kind === "class") binding.element.classList.toggle(binding.name, truthy(value));
  if (binding.kind === "attribute") writeAttribute(binding.element, binding.name, value);
  if (binding.kind === "model") writeControl(binding.element, value);
}

function readControl(element) {
  if (element.type === "checkbox") return element.checked;
  if (element.type === "radio") return element.checked ? element.value : undefined;
  if (element.type === "number" || element.type === "range") return element.valueAsNumber;
  return element.value;
}

function writeControl(element, value) {
  if (element.type === "checkbox") element.checked = truthy(value);
  else if (element.type === "radio") element.checked = element.value === String(value ?? "");
  else if (element.value !== String(value ?? "")) element.value = String(value ?? "");
}

function writeAttribute(element, name, value) {
  if (name.startsWith("aria-")) {
    element.setAttribute(name, String(Boolean(value)));
    return;
  }
  if (value === false || value === null || value === undefined) element.removeAttribute(name);
  else element.setAttribute(name, String(value));
}

function displayValue(value) {
  if (value === null || value === undefined) return "";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function truthy(value) {
  if (Array.isArray(value)) return value.length > 0;
  return Boolean(value);
}
