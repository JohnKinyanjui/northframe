let navigationSequence = 0;

export function enableNavigation() {
  document.addEventListener("click", handleClick);
  window.addEventListener("popstate", () => navigate(window.location.href, { history: "none" }));
}

async function handleClick(event) {
  if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
  const link = event.target instanceof Element ? event.target.closest("a[href]") : null;
  if (!link || !shouldNavigate(link)) return;

  const destination = new URL(link.href, window.location.href);
  if (destination.pathname === window.location.pathname && destination.search === window.location.search && destination.hash) return;

  event.preventDefault();
  await navigate(destination.href, { history: "push" });
}

function shouldNavigate(link) {
  if (link.hasAttribute("download") || link.hasAttribute("data-north-reload")) return false;
  if (link.target && link.target !== "_self") return false;
  if (link.hasAttribute("hx-get") || link.hasAttribute("data-hx-get")) return false;
  const destination = new URL(link.href, window.location.href);
  return destination.origin === window.location.origin && (destination.protocol === "http:" || destination.protocol === "https:");
}

async function navigate(destination, options) {
  const url = new URL(destination, window.location.href);
  document.documentElement.setAttribute("data-north-navigating", "true");
  document.dispatchEvent(new CustomEvent("northframe:navigation-start", { detail: { url } }));

  try {
    const response = await fetch(url, {
      credentials: "same-origin",
      headers: {
        Accept: "text/html",
        "X-Northframe-Navigation": "true",
      },
    });
    if (!response.ok || !(response.headers.get("Content-Type") || "").includes("text/html")) {
      window.location.assign(response.url || url.href);
      return;
    }

    const nextDocument = new DOMParser().parseFromString(await response.text(), "text/html");
    const replacement = routeReplacement(document, nextDocument);
    if (!replacement) {
      window.location.assign(response.url || url.href);
      return;
    }

    const scripts = Array.from(replacement.next.querySelectorAll('script[type="module"][src^="/_northframe/components/"]'))
      .map((script) => script.getAttribute("src"))
      .filter(Boolean);
    replacement.next.querySelectorAll('script[type="module"][src^="/_northframe/components/"]').forEach((script) => script.remove());
    replacement.current.dispatchEvent(new CustomEvent("northframe:before-route-swap", { bubbles: true }));
    replacement.current.replaceWith(document.importNode(replacement.next, true));

    document.title = nextDocument.title || document.title;
    if (options.history === "push") history.pushState({ northframe: true }, "", response.url || url.href);
    window.Northframe?.mount(document);
    navigationSequence += 1;
    await Promise.all(scripts.map((source) => import(`${source}${source.includes("?") ? "&" : "?"}north-nav=${navigationSequence}`)));
    document.dispatchEvent(new CustomEvent("northframe:navigation-complete", { detail: { url: new URL(response.url || url.href) } }));
    if (url.hash) document.getElementById(decodeURIComponent(url.hash.slice(1)))?.scrollIntoView();
    else window.scrollTo({ top: 0, left: 0, behavior: "instant" });
  } catch (error) {
    document.dispatchEvent(new CustomEvent("northframe:navigation-error", { detail: { url, error } }));
    window.location.assign(url.href);
  } finally {
    document.documentElement.removeAttribute("data-north-navigating");
  }
}

function routeReplacement(currentDocument, nextDocument) {
  let current = currentDocument.querySelector("[data-north-route-segment]");
  let next = nextDocument.querySelector("[data-north-route-segment]");
  if (!current || !next) return null;

  while (
    current.dataset.northRouteKind === "layout" &&
    next.dataset.northRouteKind === "layout" &&
    current.dataset.northRouteSegment === next.dataset.northRouteSegment
  ) {
    const currentChild = directRouteChild(current);
    const nextChild = directRouteChild(next);
    if (!currentChild || !nextChild) break;
    current = currentChild;
    next = nextChild;
  }
  return { current, next };
}

function directRouteChild(root) {
  for (const child of root.children) {
    if (child.hasAttribute("data-north-route-segment")) return child;
    const nested = child.querySelector("[data-north-route-segment]");
    if (nested) return nested;
  }
  return null;
}
