// Canonical layout component re-exports.
// Active components live in src/components/ for backward compat with route imports.
// Migrate here incrementally — update routes to import from @/shared/components/layout.
export { AppSidebar } from "../../components/app-sidebar";
export { ThemeToggle } from "../../components/theme-toggle";
export { NavMain } from "../../components/nav-main";
export { NavUser } from "../../components/nav-user";
