import { mountCalculators } from "./calculator.js";
import { enhanceForms } from "./forms.js";
import { enableNavigation } from "./navigation.js";
import { enableDevelopmentReload } from "./reload.js";

enhanceForms(document);
mountCalculators(document);
enableNavigation();
enableDevelopmentReload();

window.Northframe = Object.assign(window.Northframe || {}, {
  mount(root = document) {
    enhanceForms(root);
    mountCalculators(root);
  },
});
