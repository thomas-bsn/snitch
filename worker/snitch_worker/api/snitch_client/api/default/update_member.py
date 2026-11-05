from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.member_patch import MemberPatch
from ...models.status import Status
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: MemberPatch,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/admin/members/{id}".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | Status:
    if response.status_code == 200:
        response_200 = Status.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | Status]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: MemberPatch,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | Status]:
    """  Modifie les champs envoyés ; null efface le nom affiché et la langue. Le rôle, l'approbation et les
    comptes administrateurs suivent les mêmes règles que les routes dédiées ; on ne change pas son
    propre rôle ni son approbation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (MemberPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | Status]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    body: MemberPatch,
    origin: str,
    x_csrf_token: str,

) -> Error | Status | None:
    """  Modifie les champs envoyés ; null efface le nom affiché et la langue. Le rôle, l'approbation et les
    comptes administrateurs suivent les mêmes règles que les routes dédiées ; on ne change pas son
    propre rôle ni son approbation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (MemberPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | Status
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: MemberPatch,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | Status]:
    """  Modifie les champs envoyés ; null efface le nom affiché et la langue. Le rôle, l'approbation et les
    comptes administrateurs suivent les mêmes règles que les routes dédiées ; on ne change pas son
    propre rôle ni son approbation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (MemberPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | Status]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    body: MemberPatch,
    origin: str,
    x_csrf_token: str,

) -> Error | Status | None:
    """  Modifie les champs envoyés ; null efface le nom affiché et la langue. Le rôle, l'approbation et les
    comptes administrateurs suivent les mêmes règles que les routes dédiées ; on ne change pas son
    propre rôle ni son approbation.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (MemberPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | Status
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
