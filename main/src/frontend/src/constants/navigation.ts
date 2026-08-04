export interface NavigationItem {
  label: string;
  to: string;
}

export const PUBLIC_NAVIGATION_ITEMS: NavigationItem[] = [
  {
    label: "Services",
    to: "/services"
  },
  {
    label: "Contact",
    to: "/contact"
  },
  {
    label: "Feedback",
    to: "/feedback"
  },
  {
    label: "Our Vision",
    to: "/our-vision"
  }
];
