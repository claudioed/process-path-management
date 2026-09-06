import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebar: SidebarsConfig = {
  apisidebar: [
    {
      type: "doc",
      id: "api-reference/rest/process-path-management-api",
    },
    {
      type: "category",
      label: "process-paths",
      link: {
        type: "doc",
        id: "api-reference/rest/process-paths",
      },
      items: [
        {
          type: "doc",
          id: "api-reference/rest/define-path",
          label: "Define a brand-new process path",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api-reference/rest/list-paths",
          label: "List process paths",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api-reference/rest/get-path",
          label: "Get one process path by id",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api-reference/rest/revise-path",
          label: "Revise an Active process path's matchPrefix/requiredCapabilities",
          className: "api-method put",
        },
        {
          type: "doc",
          id: "api-reference/rest/deactivate-path",
          label: "Deactivate (retire) a process path",
          className: "api-method delete",
        },
      ],
    },
    {
      type: "category",
      label: "health",
      link: {
        type: "doc",
        id: "api-reference/rest/health",
      },
      items: [
        {
          type: "doc",
          id: "api-reference/rest/get-healthz",
          label: "Liveness probe",
          className: "api-method get",
        },
      ],
    },
  ],
};

export default sidebar.apisidebar;
