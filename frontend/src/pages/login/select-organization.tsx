/* eslint-disable jsx-a11y/no-static-element-interactions */
/* eslint-disable jsx-a11y/click-events-have-key-events */
import { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import Head from "next/head";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/router";
import { faArrowRight } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import axios from "axios";
import jwt_decode from "jwt-decode";

import { useNotificationContext } from "@app/components/context/Notifications/NotificationProvider";
import { IsCliLoginSuccessful } from "@app/components/utilities/attemptCliLogin";
import { Button, Spinner } from "@app/components/v2";
import { useUser } from "@app/context";
import { useGetOrganizations, useLogoutUser, useSelectOrganization } from "@app/hooks/api";
import { Organization } from "@app/hooks/api/types";
import { getAuthToken, isLoggedIn } from "@app/reactQuery";
import { navigateUserToOrg } from "@app/views/Login/Login.utils";

const LoadingScreen = () => {
  return (
    <div className="flex max-h-screen min-h-screen flex-col items-center justify-center gap-2 overflow-y-auto bg-gradient-to-tr from-mineshaft-600 via-mineshaft-800 to-bunker-700">
      <Spinner />
      <p className="text-white opacity-80">Loading, please wait</p>
    </div>
  );
};

export default function LoginPage() {
  const router = useRouter();
  const { t } = useTranslation();

  const organizations = useGetOrganizations();
  const selectOrg = useSelectOrganization();
  const { user, isLoading: userLoading } = useUser();

  const { createNotification } = useNotificationContext();

  const queryParams = new URLSearchParams(window.location.search);

  const logout = useLogoutUser(true);
  const handleLogout = useCallback(async () => {
    try {
      console.log("Logging out...");
      await logout.mutateAsync();
      router.push("/login");
    } catch (error) {
      console.error(error);
    }
  }, [logout, router]);

  const handleSelectOrganization = useCallback(
    async (organization: Organization) => {
      if (organization.authEnforced) {
        // org has an org-level auth method enabled (e.g. SAML)
        // -> logout + redirect to SAML SSO

        await logout.mutateAsync();
        window.open(`/api/v1/sso/redirect/saml2/organizations/${organization.slug}`);
        window.close();
        return;
      }

      const { token } = await selectOrg.mutateAsync({ organizationId: organization.id });

      const callbackPort = queryParams.get("callback_port");

      if (callbackPort) {
        const privateKey = localStorage.getItem("PRIVATE_KEY");

        if (!privateKey) {
          createNotification({
            text: "Private key not found",
            type: "error"
          });
        }

        if (!user.email) {
          createNotification({
            text: "User email not found",
            type: "error"
          });
        }

        if (!token) {
          createNotification({
            text: "No token found",
            type: "error"
          });
        }

        const payload = {
          JTWToken: token,
          email: user.email,
          privateKey
        } as IsCliLoginSuccessful["loginResponse"];

        // send request to server endpoint
        const instance = axios.create();
        await instance.post(`http://127.0.0.1:${callbackPort}/`, payload);
        // cli page
        router.push("/cli-redirect");
      } else {
        navigateUserToOrg(router, organization.id);
      }
    },
    [selectOrg]
  );

  useEffect(() => {
    const authToken = getAuthToken();
    const callbackPort = queryParams.get("callback_port");

    if (authToken && !callbackPort) {
      const decodedJwt = jwt_decode(authToken) as any;

      if (decodedJwt?.organizationId) {
        navigateUserToOrg(router, decodedJwt.organizationId);
      }
    }

    if (!isLoggedIn()) {
      router.push("/login");
    }
  }, [router]);

  if (userLoading || !user) {
    return <LoadingScreen />;
  }

  return (
    <div className="flex max-h-screen min-h-screen flex-col justify-center overflow-y-auto bg-gradient-to-tr from-mineshaft-600 via-mineshaft-800 to-bunker-700">
      <Head>
        <title>{t("common.head-title", { title: t("login.title") })}</title>
        <link rel="icon" href="/infisical.ico" />
        <meta property="og:image" content="/images/message.png" />
        <meta property="og:title" content={t("login.og-title") ?? ""} />
        <meta name="og:description" content={t("login.og-description") ?? ""} />
      </Head>
      <div className="mx-auto mt-20 w-fit rounded-lg border-2 border-mineshaft-500 p-10 shadow-lg">
        <Link href="/">
          <div className="mb-4 flex justify-center">
            <Image src="/images/gradientLogo.svg" height={90} width={120} alt="Infisical logo" />
          </div>
        </Link>
        <form
          onSubmit={() => console.log("submit")}
          className="mx-auto flex w-full flex-col items-center justify-center"
        >
          <div className="mb-8 space-y-2">
            <h1 className="bg-gradient-to-b from-white to-bunker-200 bg-clip-text text-center text-2xl font-medium text-transparent">
              Choose your organization
            </h1>

            <div className="space-y-1">
              <p className="text-md text-center text-gray-500">
                You&lsquo;re currently logged in as <strong>{user.email}</strong>
              </p>
              <p className="text-md text-center text-gray-500">
                Not you?{" "}
                <Button variant="link" onClick={handleLogout} className="font-semibold">
                  Change account
                </Button>
              </p>
            </div>
          </div>
          <div className="mt-2 w-1/4 min-w-[21.2rem] space-y-4 rounded-md text-center md:min-w-[25.1rem] lg:w-1/4">
            {organizations.isLoading ? (
              <Spinner />
            ) : (
              organizations.data?.map((org) => (
                <div
                  onClick={() => handleSelectOrganization(org)}
                  key={org.id}
                  className="group flex cursor-pointer items-center justify-between rounded-md bg-mineshaft-700 px-4 py-3 capitalize text-gray-200 shadow-md transition-colors hover:bg-mineshaft-600"
                >
                  <p className="transition-colors">{org.name}</p>

                  <FontAwesomeIcon
                    icon={faArrowRight}
                    className="text-gray-400 transition-colors group-hover:text-primary-500"
                  />
                </div>
              ))
            )}
          </div>

          <Button
            className="mt-8"
            colorSchema="primary"
            isLoading={logout.isLoading}
            isDisabled={logout.isLoading}
            variant="solid"
            size="lg"
            onClick={handleLogout}
          >
            Logout
          </Button>
        </form>
      </div>

      <div className="pb-28" />
    </div>
  );
}
