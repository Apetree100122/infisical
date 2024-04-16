import { TOrgPermission } from "@app/lib/types";

export type TCreateScimTokenDTO = {
  description: string;
  ttlDays: number;
} & TOrgPermission;

export type TDeleteScimTokenDTO = {
  scimTokenId: string;
} & Omit<TOrgPermission, "orgId">;

// SCIM server endpoint types

export type TListScimUsersDTO = {
  offset: number;
  limit: number;
  filter?: string;
  orgId: string;
};

export type TListScimUsers = {
  schemas: ["urn:ietf:params:scim:api:messages:2.0:ListResponse"];
  totalResults: number;
  Resources: TScimUser[];
  itemsPerPage: number;
  startIndex: number;
};

export type TGetScimUserDTO = {
  userId: string;
  orgId: string;
};

export type TCreateScimUserDTO = {
  username: string;
  email?: string;
  firstName: string;
  lastName: string;
  orgId: string;
  externalId: string;
};

export type TUpdateScimUserDTO = {
  userId: string;
  orgId: string;
  operations: {
    op: string;
    path?: string;
    value?:
      | string
      | {
          active: boolean;
        };
  }[];
};

export type TReplaceScimUserDTO = {
  userId: string;
  active: boolean;
  orgId: string;
};

export type TDeleteScimUserDTO = {
  userId: string;
  orgId: string;
};

export type TListScimGroupsDTO = {
  offset: number;
  limit: number;
  orgId: string;
};

export type TListScimGroups = {
  schemas: ["urn:ietf:params:scim:api:messages:2.0:ListResponse"];
  totalResults: number;
  Resources: TScimGroup[];
  itemsPerPage: number;
  startIndex: number;
};

export type TCreateScimGroupDTO = {
  displayName: string;
  orgId: string;
};

export type TGetScimGroupDTO = {
  groupId: string;
  orgId: string;
};

export type TUpdateScimGroupNamePutDTO = {
  groupId: string;
  orgId: string;
  displayName: string;
};

export type TUpdateScimGroupNamePatchDTO = {
  groupId: string;
  orgId: string;
  operations: (TRemoveOp | TReplaceOp | TAddOp)[];
};

type TReplaceOp = {
  op: "replace";
  value: {
    id: string;
    displayName: string;
  };
};

type TRemoveOp = {
  op: "remove";
  path: string;
};

type TAddOp = {
  op: "add";
  value: {
    value: string;
    display?: string;
  };
};

export type TDeleteScimGroupDTO = {
  groupId: string;
  orgId: string;
};

export type TScimTokenJwtPayload = {
  scimTokenId: string;
  authTokenType: string;
};

export type TScimUser = {
  schemas: string[];
  id: string;
  userName: string;
  displayName: string;
  name: {
    givenName: string;
    middleName: null;
    familyName: string;
  };
  emails: {
    primary: boolean;
    value: string;
    type: string;
  }[];
  active: boolean;
  groups: string[];
  meta: {
    resourceType: string;
    location: null;
  };
};

export type TScimGroup = {
  schemas: string[];
  id: string;
  displayName: string;
  members: {
    value: string;
    display: string;
  }[];
  meta: {
    resourceType: string;
    location: null;
  };
};
