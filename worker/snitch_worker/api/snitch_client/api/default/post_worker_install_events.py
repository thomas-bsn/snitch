from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.post_worker_install_events_body import PostWorkerInstallEventsBody
from typing import cast



def _get_kwargs(
    *,
    body: PostWorkerInstallEventsBody,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}






    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/worker/install-events",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: PostWorkerInstallEventsBody,

) -> Response[Any | Error]:
    """  Événements d'installation envoyés par le lanceur avec le jeton du Worker. Chaque événement porte
    uniquement une phase, un statut, un code de cause stable et son horodatage d'origine : aucun texte
    libre, aucun secret. Les événements mis en file hors ligne arrivent plus tard avec leur horodatage
    d'origine et sont marqués « reçus plus tard ». Un lot invalide est refusé en entier
    (invalid_install_event).

    Args:
        body (PostWorkerInstallEventsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: PostWorkerInstallEventsBody,

) -> Any | Error | None:
    """  Événements d'installation envoyés par le lanceur avec le jeton du Worker. Chaque événement porte
    uniquement une phase, un statut, un code de cause stable et son horodatage d'origine : aucun texte
    libre, aucun secret. Les événements mis en file hors ligne arrivent plus tard avec leur horodatage
    d'origine et sont marqués « reçus plus tard ». Un lot invalide est refusé en entier
    (invalid_install_event).

    Args:
        body (PostWorkerInstallEventsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return sync_detailed(
        client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: PostWorkerInstallEventsBody,

) -> Response[Any | Error]:
    """  Événements d'installation envoyés par le lanceur avec le jeton du Worker. Chaque événement porte
    uniquement une phase, un statut, un code de cause stable et son horodatage d'origine : aucun texte
    libre, aucun secret. Les événements mis en file hors ligne arrivent plus tard avec leur horodatage
    d'origine et sont marqués « reçus plus tard ». Un lot invalide est refusé en entier
    (invalid_install_event).

    Args:
        body (PostWorkerInstallEventsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: PostWorkerInstallEventsBody,

) -> Any | Error | None:
    """  Événements d'installation envoyés par le lanceur avec le jeton du Worker. Chaque événement porte
    uniquement une phase, un statut, un code de cause stable et son horodatage d'origine : aucun texte
    libre, aucun secret. Les événements mis en file hors ligne arrivent plus tard avec leur horodatage
    d'origine et sont marqués « reçus plus tard ». Un lot invalide est refusé en entier
    (invalid_install_event).

    Args:
        body (PostWorkerInstallEventsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,

    )).parsed
