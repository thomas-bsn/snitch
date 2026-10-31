from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.list_person_link_requests_status import ListPersonLinkRequestsStatus
from ...models.person_link_request_list import PersonLinkRequestList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    status: ListPersonLinkRequestsStatus | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/person-link-requests",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PersonLinkRequestList:
    if response.status_code == 200:
        response_200 = PersonLinkRequestList.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PersonLinkRequestList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    status: ListPersonLinkRequestsStatus | Unset = UNSET,

) -> Response[Error | PersonLinkRequestList]:
    """  Administrateurs uniquement. File des demandes de liaison entre un compte et une Personne.

    Args:
        status (ListPersonLinkRequestsStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonLinkRequestList]
     """


    kwargs = _get_kwargs(
        status=status,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    status: ListPersonLinkRequestsStatus | Unset = UNSET,

) -> Error | PersonLinkRequestList | None:
    """  Administrateurs uniquement. File des demandes de liaison entre un compte et une Personne.

    Args:
        status (ListPersonLinkRequestsStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonLinkRequestList
     """


    return sync_detailed(
        client=client,
status=status,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    status: ListPersonLinkRequestsStatus | Unset = UNSET,

) -> Response[Error | PersonLinkRequestList]:
    """  Administrateurs uniquement. File des demandes de liaison entre un compte et une Personne.

    Args:
        status (ListPersonLinkRequestsStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonLinkRequestList]
     """


    kwargs = _get_kwargs(
        status=status,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    status: ListPersonLinkRequestsStatus | Unset = UNSET,

) -> Error | PersonLinkRequestList | None:
    """  Administrateurs uniquement. File des demandes de liaison entre un compte et une Personne.

    Args:
        status (ListPersonLinkRequestsStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonLinkRequestList
     """


    return (await asyncio_detailed(
        client=client,
status=status,

    )).parsed
