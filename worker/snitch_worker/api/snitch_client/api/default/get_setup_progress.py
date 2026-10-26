from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.setup_progress import SetupProgress
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/setup-progress",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | SetupProgress:
    if response.status_code == 200:
        response_200 = SetupProgress.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | SetupProgress]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,

) -> Response[Error | SetupProgress]:
    """  Administrateurs et Super-administrateur : progression de l'installation guidée. L'état de chaque
    étape facultative reflète la configuration réelle (faite si elle est configurée, passée si la
    personne l'a traversée sans la configurer, à faire sinon).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SetupProgress]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,

) -> Error | SetupProgress | None:
    """  Administrateurs et Super-administrateur : progression de l'installation guidée. L'état de chaque
    étape facultative reflète la configuration réelle (faite si elle est configurée, passée si la
    personne l'a traversée sans la configurer, à faire sinon).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SetupProgress
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,

) -> Response[Error | SetupProgress]:
    """  Administrateurs et Super-administrateur : progression de l'installation guidée. L'état de chaque
    étape facultative reflète la configuration réelle (faite si elle est configurée, passée si la
    personne l'a traversée sans la configurer, à faire sinon).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SetupProgress]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,

) -> Error | SetupProgress | None:
    """  Administrateurs et Super-administrateur : progression de l'installation guidée. L'état de chaque
    étape facultative reflète la configuration réelle (faite si elle est configurée, passée si la
    personne l'a traversée sans la configurer, à faire sinon).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SetupProgress
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
