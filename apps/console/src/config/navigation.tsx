import {
  BarChart3Icon,
  BellIcon,
  FileTextIcon,
  KeyRoundIcon,
  LayoutDashboardIcon,
  MailIcon,
  ScrollTextIcon,
  ServerIcon,
  Settings2Icon,
  ShieldCheckIcon,
  UsersIcon,
  WorkflowIcon,
} from "lucide-react";
import type { ReactNode } from "react";

export type NavItem = {
  title: string;
  url: string;
  icon?: ReactNode;
  isActive?: boolean;
  items?: { title: string; url: string }[];
};

export type NavGroup = {
  label: string;
  items: NavItem[];
};

export const navGroups: NavGroup[] = [
  {
    label: "Platform",
    items: [
      {
        title: "Dashboard",
        url: "/",
        icon: <LayoutDashboardIcon />,
        isActive: true,
      },
      {
        title: "Notifications",
        url: "/logs",
        icon: <BellIcon />,
      },
      {
        title: "Analytics",
        url: "/analytics",
        icon: <BarChart3Icon />,
      },
    ],
  },
  {
    label: "Messaging",
    items: [
      {
        title: "Templates",
        url: "/templates",
        icon: <FileTextIcon />,
      },
      {
        title: "Workflows",
        url: "/workflows",
        icon: <WorkflowIcon />,
      },
      {
        title: "Subscribers",
        url: "/subscribers",
        icon: <UsersIcon />,
      },
    ],
  },
  {
    label: "Settings",
    items: [
      {
        title: "Settings",
        url: "/settings",
        icon: <Settings2Icon />,
        items: [
          { title: "Providers", url: "/settings/providers" },
          { title: "India DLT", url: "/settings/dlt" },
          { title: "API Keys", url: "/settings/api-keys" },
        ],
      },
    ],
  },
];
