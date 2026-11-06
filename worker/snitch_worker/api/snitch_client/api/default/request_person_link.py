from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.person_link_request import PersonLinkRequest
from ...models.person_link_request_input import PersonLinkRequestInput
from typing import cast



def _get_kwargs(
    *,
    body: PersonLinkRequestInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/me/person/request",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PersonLinkRequest:
    if response.status_code == 201:
        response_201 = PersonLinkRequest.from_dict(response.json())



        return response_201

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PersonLinkRequest]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: PersonLinkRequestInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PersonLinkRequest]:
    """  Utilisateurs connectés. Demande à lier son compte à une Personne existante ; un Administrateur
    approuve ou refuse. Un compte déjà lié doit se délier d'abord. Un compte a au plus une demande en
    attente, et une Personne aussi.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PersonLinkRequestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonLinkRequest]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: PersonLinkRequestInput,
    origin: str,
    x_csrf_token: str,

) -> Error | PersonLinkRequest | None:
    """  Utilisateurs connectés. Demande à lier son compte à une Personne existante ; un Administrateur
    approuve ou refuse. Un compte déjà lié doit se délier d'abord. Un compte a au plus une demande en
    attente, et une Personne aussi.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PersonLinkRequestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonLinkRequest
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: PersonLinkRequestInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PersonLinkRequest]:
    """  Utilisateurs connectés. Demande à lier son compte à une Personne existante ; un Administrateur
    approuve ou refuse. Un compte déjà lié doit se délier d'abord. Un compte a au plus une demande en
    attente, et une Personne aussi.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PersonLinkRequestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonLinkRequest]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: PersonLinkRequestInput,
    origin: str,
    x_csrf_token: str,

) -> Error | PersonLinkRequest | None:
    """  Utilisateurs connectés. Demande à lier son compte à une Personne existante ; un Administrateur
    approuve ou refuse. Un compte déjà lié doit se délier d'abord. Un compte a au plus une demande en
    attente, et une Personne aussi.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PersonLinkRequestInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonLinkRequest
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
