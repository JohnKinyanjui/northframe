const originalDisabled = new WeakMap();

export function enhanceForms(root = document) {
  const csrfToken = readCookie("northframe_csrf");
  for (const form of root.querySelectorAll("form")) {
    addCSRFToken(form, csrfToken);
    setPending(form, false);
    if (!form.hasAttribute("nf-enhance") || form.dataset.nfEnhanced === "true") continue;
    form.dataset.nfEnhanced = "true";
    form.addEventListener("submit", (event) => submitEnhancedForm(event, form, csrfToken));
  }
}

async function submitEnhancedForm(event, form, csrfToken) {
  const submitter = event.submitter;
  const methodSource = submitter?.hasAttribute("formmethod")
    ? submitter.getAttribute("formmethod")
    : form.method;
  const actionSource = submitter?.hasAttribute("formaction")
    ? submitter.getAttribute("formaction")
    : form.action;
  const method = (methodSource || "get").toUpperCase();
  const action = new URL(actionSource || window.location.href, window.location.href);
  if (method !== "POST" || action.origin !== window.location.origin) return;

  event.preventDefault();
  const data = formDataFor(form, submitter);
  clearActionResult(form);
  setPending(form, true);
  dispatch(form, "northframe:submit", { action, data, submitter });

  try {
    const response = await fetch(action, {
      method,
      body: data,
      credentials: "same-origin",
      headers: {
        Accept: "text/html, application/json",
        "X-Northframe-Enhance": "true",
        ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
      },
    });
    if (response.redirected) {
      dispatch(form, "northframe:success", { response });
      window.location.assign(response.url);
      return;
    }
    const contentType = response.headers.get("Content-Type") || "";
    if (contentType.includes("text/html")) {
      const html = await response.text();
      dispatch(form, response.ok ? "northframe:success" : "northframe:invalid", { response });
      document.open();
      document.write(html);
      document.close();
      return;
    }
    if (contentType.includes("application/json")) {
      const result = await response.json();
      applyActionResult(form, result);
      dispatch(form, "northframe:result", { response, result });
      dispatch(form, response.ok && result.success ? "northframe:success" : "northframe:invalid", { response, result });
      if (result.redirect) window.location.assign(new URL(result.redirect, action));
      return;
    }
    dispatch(form, response.ok ? "northframe:success" : "northframe:invalid", { response, result: null });
  } catch (error) {
    dispatch(form, "northframe:error", { error, source: "form" });
  } finally {
    if (document.contains(form)) {
      setPending(form, false);
      dispatch(form, "northframe:complete", {});
    }
  }
}

function clearActionResult(form) {
  for (const control of form.querySelectorAll("[aria-invalid='true']")) {
    control.removeAttribute("aria-invalid");
    control.removeAttribute("data-nf-invalid");
    const errorID = control.getAttribute("data-nf-error-describedby");
    if (errorID) {
      const describedBy = (control.getAttribute("aria-describedby") || "").split(/\s+/).filter((id) => id && id !== errorID);
      if (describedBy.length) control.setAttribute("aria-describedby", describedBy.join(" "));
      else control.removeAttribute("aria-describedby");
      control.removeAttribute("data-nf-error-describedby");
    }
  }
  for (const target of form.querySelectorAll("[nf-error]")) {
    target.textContent = "";
    target.hidden = true;
  }
  for (const target of form.querySelectorAll("[nf-message]")) {
    target.textContent = "";
    target.hidden = true;
  }
}

function applyActionResult(form, result) {
  clearActionResult(form);
  for (const target of form.querySelectorAll("[nf-message]")) {
    target.textContent = result.message || "";
    target.hidden = !result.message;
  }
  let firstInvalid = null;
  for (const [name, message] of Object.entries(result.errors || {})) {
    const control = form.elements.namedItem(name);
    const controls = control instanceof RadioNodeList ? Array.from(control) : control ? [control] : [];
    for (const current of controls) {
      if (!(current instanceof HTMLElement)) continue;
      current.setAttribute("aria-invalid", "true");
      current.setAttribute("data-nf-invalid", "true");
      if (!firstInvalid) firstInvalid = current;
    }
    for (const target of form.querySelectorAll(`[nf-error="${escapeSelectorValue(name)}"]`)) {
      target.textContent = String(message);
      target.hidden = false;
      if (target.id) {
        for (const current of controls) {
          if (!(current instanceof HTMLElement)) continue;
          const describedBy = new Set((current.getAttribute("aria-describedby") || "").split(/\s+/).filter(Boolean));
          describedBy.add(target.id);
          current.setAttribute("aria-describedby", Array.from(describedBy).join(" "));
          current.setAttribute("data-nf-error-describedby", target.id);
        }
      }
    }
  }
  firstInvalid?.focus();
}

function escapeSelectorValue(value) {
  return String(value).replaceAll("\\", "\\\\").replaceAll('"', '\\"');
}

function setPending(form, pending) {
  form.classList.toggle("nf-pending", pending);
  form.setAttribute("aria-busy", String(pending));
  for (const element of form.querySelectorAll("[nf-loading]")) element.hidden = !pending;
  for (const element of form.querySelectorAll("[nf-idle]")) element.hidden = pending;
  for (const control of form.querySelectorAll('button[type="submit"], button:not([type]), input[type="submit"]')) {
    if (pending) {
      originalDisabled.set(control, control.disabled);
      control.disabled = true;
    } else if (originalDisabled.has(control)) {
      control.disabled = originalDisabled.get(control);
      originalDisabled.delete(control);
    }
  }
}

function addCSRFToken(form, token) {
  const method = (form.getAttribute("method") || "get").toLowerCase();
  if (!token || method === "get" || form.querySelector('input[name="_northframe_csrf"]')) return;
  const input = document.createElement("input");
  input.type = "hidden";
  input.name = "_northframe_csrf";
  input.value = token;
  form.append(input);
}

function formDataFor(form, submitter) {
  try {
    return new FormData(form, submitter);
  } catch {
    const data = new FormData(form);
    if (submitter?.name) data.append(submitter.name, submitter.value);
    return data;
  }
}

function readCookie(name) {
  const prefix = `${encodeURIComponent(name)}=`;
  const entry = document.cookie.split(";").map((value) => value.trim()).find((value) => value.startsWith(prefix));
  return entry ? decodeURIComponent(entry.slice(prefix.length)) : "";
}

function dispatch(form, name, detail) {
  form.dispatchEvent(new CustomEvent(name, { bubbles: true, detail }));
  if (name.startsWith("northframe:")) {
    form.dispatchEvent(new CustomEvent(name.replace(":", "-"), { bubbles: true, detail }));
  }
}
