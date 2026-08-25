import { mountCalculators } from "./calculator.js";
import { enhanceForms } from "./forms.js";
import { enableDevelopmentReload } from "./reload.js";

enhanceForms(document);
mountCalculators(document);
enableDevelopmentReload();

window.Northframe = Object.assign(window.Northframe || {}, {
  mount(root = document) {
    enhanceForms(root);
    mountCalculators(root);
  },
});
