import { TOrgRole } from "../roles/types";

export type TGroupOrgMembership = TGroup & {
  customRole?: TOrgRole;
}

export type TGroup = {
  id: string;
  name: string;
  slug: string;
  orgId: string;
  createAt: string;
  updatedAt: string;
  role: string;
};