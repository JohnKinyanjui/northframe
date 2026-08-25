import { mountCalculators } from "./calculator.js";
import { enhanceForms } from "./forms.js";

enhanceForms(document);
mountCalculators(document);

window.Northframe = Object.assign(window.Northframe || {}, {
  mount(root = document) {
    enhanceForms(root);
    mountCalculators(root);
  },
});
